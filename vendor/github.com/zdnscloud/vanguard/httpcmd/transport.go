package httpcmd

import (
	"encoding/json"
	"net/http"

	"github.com/zdnscloud/cement/rest"
	"github.com/zdnscloud/vanguard/metrics"
)

type HttpTransport struct {
}

func NewHttpTransport() *HttpTransport {
	return &HttpTransport{}
}

func (t *HttpTransport) Run(s Service, p *HttpCmdProtocol, e *EndPoint) {
	handler := func(req *http.Request) (int, string) {
		task, err := p.DecodeTask(req)
		if err != nil {
			errBody, _ := json.Marshal(err)
			return int(InnerError), string(errBody)
		}

		result, err := p.EncodeTaskResult(s.HandleTask(task))
		if err != nil {
			errBody, _ := json.Marshal(err)
			return int(InnerError), string(errBody)
		}

		return result.Code, result.Body
	}

	healthzHandler := func(req *http.Request) (int, string) {
		return int(Succeed), "OK"
	}

	server := rest.NewRestServer()
	server.RegisterHandler("/"+e.Name, handler)
	server.RegisterHandler("/health", healthzHandler)
	server.RegisterRawHandler("/metrics", metrics.Handler())
	server.Run(e.IP, e.Port)
}
