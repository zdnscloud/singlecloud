package httpcmd

type Service interface {
	SupportedCmds() []Command
	HandleTask(*Task) *TaskResult
}

func Run(s Service, e *EndPoint) error {
	p, err := NewHttpCmdProtocol(s.SupportedCmds(), e)
	if err != nil {
		return err
	}

	NewHttpTransport().Run(s, p, e)
	return nil
}
