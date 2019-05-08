package server

import (
	"github.com/zdnscloud/vanguard/acl"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/metrics"
)

type Reconfig struct {
}

func (c *Reconfig) String() string {
	return "reconfig"
}

type Ping struct {
}

func (c *Ping) String() string {
	return "ping"
}

type Stop struct {
}

func (c *Stop) String() string {
	return "stop"
}

func (s *Server) stop() {
	close(s.stopChan)
	s.wg.Wait()
	s.stopChan = make(chan struct{})
}

func (s *Server) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch cmd.(type) {
	case *Reconfig:
		metrics.Stop()
		acl.GetAclManager().Stop()
		s.stop()
		s.conf.Reload()
		acl.GetAclManager().ReloadConfig(s.conf)
		h := s.queryHandler
		for h != nil {
			if owner, ok := h.(config.ConfigureOwner); ok {
				owner.ReloadConfig(s.conf)
			}
			h = h.Next()
		}

		go metrics.Run()
		s.startHandlerRoutine(s.handlerRoutineCount)
		return nil, nil
	case *Stop:
		s.stop()
		return nil, nil
	case *Ping:
		return nil, nil
	default:
		panic("shouldn't be here")
	}
}
