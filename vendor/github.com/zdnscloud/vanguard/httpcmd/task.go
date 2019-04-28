package httpcmd

import (
	"bytes"
)

type StatusCode int

const (
	Succeed    StatusCode = 200
	InnerError            = 500
	NotAuth               = 401
)

type Command interface {
	String() string
}

type Task struct {
	Cmds []Command
}

type TaskResult struct {
	Code   StatusCode
	Result interface{}
}

func NewTask() *Task {
	return &Task{
		Cmds: make([]Command, 0),
	}
}

func (t *Task) ClearCmd() {
	t.Cmds = make([]Command, 0)
}

func (t *Task) AddCmd(c Command) {
	t.Cmds = append(t.Cmds, c)
}

func (t *Task) Failed(err *Error) *TaskResult {
	return t.FailedWithStatus(err, InnerError)
}

func (t *Task) FailedWithStatus(err *Error, code StatusCode) *TaskResult {
	return &TaskResult{
		Code:   code,
		Result: err,
	}
}

func (t *Task) Succeed() *TaskResult {
	return &TaskResult{
		Code:   Succeed,
		Result: nil,
	}
}

func (t *Task) SucceedWithResult(result interface{}) *TaskResult {
	return &TaskResult{
		Code:   Succeed,
		Result: result,
	}
}

func (t *Task) String() string {
	var info bytes.Buffer
	info.WriteString("task :[")
	for _, c := range t.Cmds {
		info.WriteString(c.String())
		info.WriteString(",\n")
	}
	info.WriteByte(']')
	return info.String()
}

func (r *TaskResult) IsSucceed() bool {
	return r.Code == Succeed
}
