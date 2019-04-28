package util

import (
	"errors"
	"net"
	"time"

	"github.com/zdnscloud/g53"
	gutil "github.com/zdnscloud/g53/util"
)

var errMalformedResponse = errors.New("response format error")

type UDPSender struct {
	dialer  *net.Dialer
	timeout time.Duration
}

func NewUDPSender(querySource string, timeout time.Duration) (*UDPSender, error) {
	sender := &UDPSender{
		dialer: &net.Dialer{
			Timeout: timeout,
		},
		timeout: timeout,
	}

	if err := sender.setQuerySource(querySource); err != nil {
		return nil, err
	} else {
		return sender, nil
	}
}

func (f *UDPSender) setQuerySource(source string) error {
	if source == "" {
		f.dialer.LocalAddr = nil
	} else {
		localAddr, err := net.ResolveUDPAddr("udp", source)
		if err != nil {
			return err
		}
		f.dialer.LocalAddr = localAddr
	}
	return nil
}

func (f *UDPSender) GetQuerySource() string {
	if f.dialer.LocalAddr == nil {
		return ""
	} else {
		return f.dialer.LocalAddr.String()
	}
}

func (f *UDPSender) SendQuery(server string, render *g53.MsgRender, query *g53.Message) (*net.UDPConn, error) {
	query.Rend(render)
	c, err := f.dialer.Dial("udp", server)
	if err != nil {
		return nil, err
	}
	conn := c.(*net.UDPConn)
	conn.SetWriteDeadline(time.Now().Add(f.timeout))
	conn.Write(render.Data())
	render.Clear()
	return conn, nil
}

func (f *UDPSender) Query(server string, render *g53.MsgRender, query *g53.Message) (*g53.Message, time.Duration, error) {
	conn, err := f.SendQuery(server, render, query)
	if err != nil {
		return nil, f.timeout, err
	}
	defer conn.Close()

	sendTime := time.Now()
	conn.SetReadDeadline(sendTime.Add(f.timeout))
	buf := make([]byte, 1024)

retry:
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, f.timeout, err
	}

	buffer := gutil.NewInputBuffer(buf[0:n])
	msg, err := g53.MessageFromWire(buffer)
	if err != nil {
		return nil, f.timeout, err
	} else if msg.Header.Id == query.Header.Id {
		rtt := time.Now().Sub(sendTime)
		if err := isResponseValid(query, msg); err != nil {
			return nil, rtt, err
		} else {
			return msg, rtt, nil
		}
	} else {
		buf = buf[:0]
		goto retry
	}
}

func isResponseValid(req *g53.Message, resp *g53.Message) error {
	if resp.Header.Rcode == g53.R_FORMERR {
		return nil
	}

	if resp.Question == nil || resp.Question.Equals(req.Question) == false {
		return errMalformedResponse
	} else {
		return nil
	}
}
