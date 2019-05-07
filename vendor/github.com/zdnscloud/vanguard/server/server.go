package server

import (
	"net"
	"runtime/debug"
	"sync"

	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/g53/util"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/logger"
	"github.com/zdnscloud/vanguard/metrics"
)

const (
	defaultHandlerCount = 1024
)

type message struct {
	usingTCP bool
	addr     net.Addr
	destAddr net.Addr
	conn     net.Conn
	buf      []byte
}

type Server struct {
	conf         *config.VanguardConf
	transport    *Transport
	queryHandler core.DNSQueryHandler
	xfrHander    core.DNSQueryHandler
	messageChan  chan message

	handlerRoutineCount int
	stopChan            chan struct{}
	wg                  sync.WaitGroup
}

func NewServer(conf *config.VanguardConf, queryHandler core.DNSQueryHandler, xfrHander core.DNSQueryHandler) (*Server, error) {
	handlerCount := conf.Server.HandlerCount
	if handlerCount == 0 {
		handlerCount = defaultHandlerCount
	}

	transport, err := newTransport(conf, handlerCount)
	if err != nil {
		return nil, err
	}

	s := &Server{
		conf:                conf,
		transport:           transport,
		messageChan:         make(chan message, handlerCount),
		queryHandler:        queryHandler,
		xfrHander:           xfrHander,
		handlerRoutineCount: handlerCount,
		stopChan:            make(chan struct{}),
	}

	httpcmd.RegisterHandler(s, []httpcmd.Command{&Reconfig{}, &Stop{}, &Ping{}})

	return s, nil
}

func (s *Server) Run() {
	s.startHandlerRoutine(s.handlerRoutineCount)
	s.transport.run(s.messageChan)
}

func (s *Server) Shutdown() {
	s.transport.Close()
	s.stop()
}

func (s *Server) startHandlerRoutine(handlerCount int) {
	for i := 0; i < handlerCount; i++ {
		s.wg.Add(1)
		go func() {
			defer func() {
				if p := recover(); p != nil {
					logger.GetLogger().Error("handler crashed caused by %v, %s", p, string(debug.Stack()))
					s.startHandlerRoutine(1)
				}
				s.wg.Done()
			}()

			ctx := core.NewContext()
			render := g53.NewMsgRender()
			inputBuff := util.NewInputBuffer(nil)
			var request g53.Message
			for {
				select {
				case <-s.stopChan:
					return
				case message := <-s.messageChan:
					inputBuff.SetData(message.buf)
					err := request.FromWire(inputBuff)
					if err == nil {
						ctx.Reset()
						ctx.Client.Addr = message.addr
						ctx.Client.DestAddr = message.destAddr
						ctx.Client.Request = &request
						ctx.Client.UsingTCP = message.usingTCP
						if request.Header.Opcode == g53.OP_QUERY {
							s.queryHandler.HandleQuery(ctx)
						} else if request.Header.Opcode == g53.OP_NOTIFY && s.xfrHander != nil {
							s.xfrHander.HandleQuery(ctx)
						} else {
							logger.GetLogger().Error("invalid opcode")
						}
						metrics.RecordMetrics(ctx.Client)
						if ctx.Client.Response != nil {
							ctx.Client.Response.RecalculateSectionRRCount()
							ctx.Client.Response.Rend(render)
							s.transport.SendResponse(&message, render.Data())
							render.Clear()
						}
					} else {
						logger.GetLogger().Error("get invalid query %s", err.Error())
					}
					s.transport.FinishQuery(&message)
				}
			}
		}()
	}
}
