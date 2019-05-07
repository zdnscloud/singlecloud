package httpcmd

import (
	"time"

	"github.com/zdnscloud/cement/rest"
)

type HttpCmdProxy struct {
	protocol *HttpCmdProtocol
	client   *rest.RestClient
}

func NewHttpCmdProxy(p *HttpCmdProtocol, e *EndPoint) (*HttpCmdProxy, error) {
	client, err := rest.NewRestClient(e.GenerateServiceUrl(), time.Minute)
	if err != nil {
		return nil, err
	}

	return &HttpCmdProxy{
		protocol: p,
		client:   client,
	}, nil
}

func (p *HttpCmdProxy) HandleTask(t *Task, succeed interface{}) *Error {
	if err := p.client.Connect(); err != nil {
		return NewError(0, err.Error())
	}

	req, err := p.protocol.EncodeTask(t)
	if err != nil {
		return err
	}
	response, _ := p.client.Send(req)
	if response == nil || response.Body == nil {
		if err := p.client.ReConnect(); err != nil {
			return NewError(0, err.Error())
		}

		//the failed send will modify req, so we reconstruct the req
		req, _ := p.protocol.EncodeTask(t)
		var err error
		response, err = p.client.Send(req)
		if response == nil || response.Body == nil {
			return NewError(0, err.Error())
		}
	}

	return p.protocol.DecodeTaskResult(response, succeed)
}

func (p *HttpCmdProxy) Close() {
	p.client.Close()
}

func GetProxy(e *EndPoint, cmds []Command) (*HttpCmdProxy, error) {
	if p, err := NewHttpCmdProtocol(cmds, e); err != nil {
		return nil, err
	} else {
		return NewHttpCmdProxy(p, e)
	}
}
