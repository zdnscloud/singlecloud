package httpcmd

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/zdnscloud/cement/serializer"
)

type EndPoint struct {
	Name string
	IP   string
	Port int
}

func (e *EndPoint) GenerateServiceUrl() string {
	return "http://" + e.IP + ":" + strconv.Itoa(e.Port) + "/" + e.Name
}

type HttpCmdProtocol struct {
	serializer *serializer.Serializer
	endPoint   EndPoint
}

func NewHttpCmdProtocol(cmds []Command, e *EndPoint) (*HttpCmdProtocol, error) {
	s := serializer.NewSerializer()
	for _, c := range cmds {
		if err := s.Register(c); err != nil {
			return nil, err
		}
	}
	return &HttpCmdProtocol{
		serializer: s,
		endPoint:   *e,
	}, nil
}

func (p *HttpCmdProtocol) DecodeTask(req interface{}) (*Task, *Error) {
	r, _ := req.(*http.Request)
	if r.Method != "POST" {
		return nil, ErrHTTPMethodInvalid
	}

	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return nil, NewError(0, err.Error())
	}

	var tt struct {
		CmdType string          `json:"resource_type"`
		Parmas  json.RawMessage `json:"attrs"`
	}

	err = json.Unmarshal(body, &tt)
	if err != nil {
		return nil, ErrCmdFormatInvalid
	}

	t := NewTask()
	cmd_, err := p.serializer.DecodeType(tt.CmdType, tt.Parmas)
	if err != nil {
		return nil, ErrUnknownCmd
	}

	c, _ := cmd_.(Command)
	t.AddCmd(c)

	return t, nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error {
	return nil
}

func (p *HttpCmdProtocol) EncodeTask(t *Task) (*http.Request, *Error) {
	r := new(http.Request)
	r.ProtoMajor = 1
	r.ProtoMinor = 1
	r.Header = map[string][]string{
		"Content-Type": {"application/json;charset=utf-8"},
		"Accept":       {"*/*"},
	}
	r.Method = "POST"
	uri := (&p.endPoint).GenerateServiceUrl()

	var tt struct {
		CmdType string      `json:"resource_type"`
		Params  interface{} `json:"attrs"`
	}

	if len(t.Cmds) != 1 {
		return nil, ErrBatchCmdNotSupport
	}
	c := t.Cmds[0]
	tt.CmdType = serializer.ObjName(c)
	tt.Params = c

	body, err := json.Marshal(tt)
	if err != nil {
		return nil, ErrCmdFormatInvalid
	}

	r.Body = nopCloser{bytes.NewBufferString(string(body))}
	r.URL, err = url.Parse(uri)
	if err != nil {
		return nil, NewError(0, err.Error())
	} else {
		return r, nil
	}
}

type HttpResult struct {
	Code int
	Body string
}

func (p *HttpCmdProtocol) EncodeTaskResult(r *TaskResult) (*HttpResult, *Error) {
	body, err := json.Marshal(r.Result)
	if err != nil {
		return nil, NewError(0, err.Error())
	}

	return &HttpResult{
		Code: int(r.Code),
		Body: string(body),
	}, nil
}

func (p *HttpCmdProtocol) DecodeTaskResult(resp interface{}, success interface{}) *Error {
	response, _ := resp.(*http.Response)
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return NewError(0, err.Error())
	}

	if response.StatusCode == int(Succeed) {
		if success != nil {
			err = json.Unmarshal([]byte(body), success)
			if err != nil {
				return NewError(0, err.Error())
			}
		}
		return nil
	}

	var executeErr Error
	err = json.Unmarshal([]byte(body), &executeErr)
	if err != nil {
		return NewError(0, err.Error())
	}
	return &executeErr
}
