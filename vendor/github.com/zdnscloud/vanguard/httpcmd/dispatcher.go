package httpcmd

import (
	"github.com/zdnscloud/cement/reflector"
	"github.com/zdnscloud/vanguard/logger"
)

type CmdHandler interface {
	HandleCmd(Command) (interface{}, *Error)
}

type HandlerOwner interface {
	RegisterHandler(*CmdDispatcher)
}

type CmdDispatcher struct {
	cmdHandlers   map[string]CmdHandler
	supportedCmds []Command
}

func newCmdDispatcher() *CmdDispatcher {
	return &CmdDispatcher{
		cmdHandlers: make(map[string]CmdHandler),
	}
}

func (dispatcher *CmdDispatcher) RegisterHandler(handler CmdHandler, supportedCmds []Command) {
	var cmdNames []string
	for _, cmd := range supportedCmds {
		handler_ := dispatcher.getHandler(cmd)
		if handler_ != nil {
			logger.GetLogger().Warn("command %s register by different handler\n", cmd.String())
		}
		cmdName, _ := reflector.StructName(cmd)
		cmdNames = append(cmdNames, cmdName)
	}

	dispatcher.supportedCmds = append(dispatcher.supportedCmds, supportedCmds...)
	for _, cmdName := range cmdNames {
		dispatcher.cmdHandlers[cmdName] = handler
	}
}

func (dispatcher *CmdDispatcher) getHandler(cmd Command) CmdHandler {
	cmdName, _ := reflector.StructName(cmd)
	handler, ok := dispatcher.cmdHandlers[cmdName]
	if ok {
		return handler
	} else {
		return nil
	}
}

//for test purpose
func (dispatcher *CmdDispatcher) ClearHandler() {
	dispatcher.cmdHandlers = make(map[string]CmdHandler)
}
