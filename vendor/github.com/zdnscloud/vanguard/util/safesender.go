package util

import (
	"sync"
	"time"

	"github.com/zdnscloud/g53"
)

type SafeUDPSender struct {
	sender     *UDPSender
	renders    []*g53.MsgRender
	renderLock sync.Mutex
}

func NewSafeUDPSender(querySource string, timeout time.Duration) (*SafeUDPSender, error) {
	sender, err := NewUDPSender(querySource, timeout)
	if err != nil {
		return nil, err
	}

	return &SafeUDPSender{
		sender:  sender,
		renders: []*g53.MsgRender{},
	}, nil
}

func (f *SafeUDPSender) GetQuerySource() string {
	return f.sender.GetQuerySource()
}

func (f *SafeUDPSender) Query(server string, query *g53.Message) (*g53.Message, time.Duration, error) {
	render := f.getRender()
	resp, rtt, err := f.sender.Query(server, render, query)
	f.releaseRender(render)
	return resp, rtt, err
}

func (f *SafeUDPSender) getRender() *g53.MsgRender {
	f.renderLock.Lock()
	defer f.renderLock.Unlock()
	for i, render := range f.renders {
		if render != nil {
			f.renders[i] = nil
			return render
		}
	}

	return g53.NewMsgRender()
}

func (f *SafeUDPSender) releaseRender(render *g53.MsgRender) {
	f.renderLock.Lock()
	defer f.renderLock.Unlock()
	for i, render_ := range f.renders {
		if render_ == nil {
			f.renders[i] = render
			return
		}
	}
	f.renders = append(f.renders, render)
}
