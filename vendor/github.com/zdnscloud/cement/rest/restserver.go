package rest

import (
	"github.com/zdnscloud/cement/httprouter"
	"net/http"
	"strconv"
)

type RestServer struct {
	m *httprouter.Router
}

func NewRestServer() *RestServer {
	return &RestServer{httprouter.New()}
}

func (s *RestServer) RegisterHandler(req string, handler func(r *http.Request) (int, string)) {
	h := func(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		status, result := handler(req)
		res.WriteHeader(status)
		res.Write([]byte(result))
	}

	s.m.GET(req, h)
	s.m.PUT(req, h)
	s.m.POST(req, h)
	s.m.DELETE(req, h)
	s.m.PATCH(req, h)
}

func (s *RestServer) RegisterRawHandler(req string, handler func(http.ResponseWriter, *http.Request)) {
	h := func(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		handler(res, req)
	}
	s.m.GET(req, h)
	s.m.PUT(req, h)
	s.m.POST(req, h)
	s.m.DELETE(req, h)
	s.m.PATCH(req, h)
}

func (s *RestServer) Run(ip string, port int) {
	ipAndPort := ip + ":" + strconv.Itoa(port)
	http.ListenAndServe(ipAndPort, s.m)
}
