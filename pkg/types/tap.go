package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Tap struct {
	Source          TcpAddress   `json:"source,omitempty"`
	SourceMeta      EndpointMeta `json:"sourceMeta,omitempty"`
	Destination     TcpAddress   `json:"destination,omitempty"`
	DestinationMeta EndpointMeta `json:"destinationMeta,omitempty"`
	RouteMeta       EndpointMeta `json:"routeMeta,omitempty"`
	ProxyDirection  string       `json:"proxyDirection,omitempty"`
	Event           Event        `json:"event,omitempty"`
}

type TcpAddress struct {
	Ip   string `json:"ip,omitempty"`
	Port int    `json:"port,omitempty"`
}

type EndpointMeta struct {
	Labels map[string]string `json:"labels,omitempty"`
}

type Event struct {
	RequestInit  HttpRequestInit  `json:requestInit,omitempty"`
	ResponseInit HttpResponseInit `json:responseInit,omitempty"`
	ResponseEnd  HttpResponseEnd  `json:responseEnd,omitempty"`
}

type HttpRequestInit struct {
	Id        HttpStreamId `json:"id,omitempty"`
	Method    string       `json:"method,omitempty"`
	Scheme    string       `json:"scheme,omitempty"`
	Authority string       `json:"authority,omitempty"`
	Path      string       `json:"path,omitempty"`
	Headers   []Header     `json:"headers,omitempty"`
}

type HttpStreamId struct {
	Base   int `json:"base,omitempty"`
	Stream int `json:"stream,omitempty"`
}

type Header struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type HttpResponseInit struct {
	Id               HttpStreamId `json:"id,omitempty"`
	SinceRequestInit Duration     `json:"sinceRequestInit,omitempty"`
	HttpStatus       int          `json:"httpStatus,omitempty"`
	Headers          []Header     `json:"headers,omitempty"`
}

type Duration struct {
	Seconds int `json:"seconds,omitempty"`
	Nanos   int `json:"nanos,omitempty"`
}

type HttpResponseEnd struct {
	Id                HttpStreamId `json:"id,omitempty"`
	SinceRequestInit  Duration     `json:"sinceRequestInit,omitempty"`
	SinceResponseInit Duration     `json:"sinceResponseInit,omitempty"`
	ResponseBytes     int          `json:"responseBytes,omitempty"`
	Eos               int          `json:"eos,omitempty"`
	Trailers          []Header     `json:"trailers,omitempty"`
}

func (t Tap) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
