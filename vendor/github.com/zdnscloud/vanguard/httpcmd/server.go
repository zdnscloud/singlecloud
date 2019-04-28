package httpcmd

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/logger"
)

type cmdService struct {
	conf       *config.VanguardConf
	handleLock sync.Mutex
}

var globalDispatcher *CmdDispatcher

func init() {
	globalDispatcher = newCmdDispatcher()
}

func RegisterHandler(handler CmdHandler, supportedCmds []Command) {
	globalDispatcher.RegisterHandler(handler, supportedCmds)
}

func ClearHandler() {
	globalDispatcher.ClearHandler()
}

func NewCmdService(conf *config.VanguardConf) *cmdService {
	return &cmdService{
		conf: conf,
	}
}

func (s *cmdService) HandleTask(t *Task) *TaskResult {
	if len(t.Cmds) != 1 {
		return t.Failed(ErrBatchCmdNotSupport)
	}

	c := t.Cmds[0]
	handler := globalDispatcher.getHandler(c)
	if handler == nil {
		return t.Failed(ErrUnknownCmd)
	}

	s.handleLock.Lock()
	result, err := s.safeRunCommand(handler, c)
	s.handleLock.Unlock()

	if err != nil {
		logger.GetLogger().Error("command %s failed: %s\n", c.String(), err.Error())
		return t.Failed(err)
	} else if result != nil {
		return t.SucceedWithResult(result)
	} else {
		return t.Succeed()
	}
}

func (s *cmdService) safeRunCommand(handler CmdHandler, c Command) (result interface{}, err *Error) {
	defer func() {
		if p := recover(); p != nil {
			result = nil
			err = ErrAssertFailed.AddDetail(fmt.Sprintf("%v", p))
		}
	}()

	result, err = handler.HandleCmd(c)
	return
}

func (s *cmdService) SupportedCmds() []Command {
	return globalDispatcher.supportedCmds
}

func (s *cmdService) Run() error {
	ipAndPort := strings.Split(s.conf.Server.HttpCmdAddr, ":")
	ip := ipAndPort[0]
	port, _ := strconv.Atoi(ipAndPort[1])
	e := &EndPoint{
		Name: "vanguard_cmd",
		IP:   ip,
		Port: port,
	}
	return Run(s, e)
}
