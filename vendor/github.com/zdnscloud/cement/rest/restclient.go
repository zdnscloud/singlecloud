package rest

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type RestClient struct {
	conn     *httputil.ClientConn
	resource *url.URL
	timeout  time.Duration
	tcpConn  *net.TCPConn
}

func NewRestClient(resource string, timeout time.Duration) (*RestClient, error) {
	client := new(RestClient)

	u, err := url.Parse(resource)
	if err != nil {
		return nil, err
	} else {
		client.resource = u
		client.timeout = timeout
	}
	return client, nil
}

func (client *RestClient) Connect() error {
	if client.conn == nil {
		return client.ReConnect()
	} else {
		return nil
	}
}

func (client *RestClient) ReConnect() error {
	if client.conn != nil {
		client.conn.Close()
	}

	conn, err := net.Dial("tcp", client.resource.Host)
	if err != nil {
		return err
	}

	tcpConn, _ := conn.(*net.TCPConn)
	if err = tcpConn.SetKeepAlive(true); err != nil {
		return err
	}

	if err = tcpConn.SetKeepAlivePeriod(time.Second); err != nil {
		return err
	}

	client.conn = httputil.NewClientConn(tcpConn, nil)
	client.tcpConn = tcpConn
	return nil
}

func (client *RestClient) Close() {
	if client.conn != nil {
		client.conn.Close()
	}
}

func (client *RestClient) Send(request *http.Request) (*http.Response, error) {
	client.tcpConn.SetWriteDeadline(time.Now().Add(client.timeout))
	err := client.conn.Write(request)
	if err != nil {
		return nil, err
	}

	client.tcpConn.SetReadDeadline(time.Now().Add(client.timeout))
	if r, err := client.conn.Read(request); err != nil {
		return r, err
	} else {
		return r, nil
	}
}
