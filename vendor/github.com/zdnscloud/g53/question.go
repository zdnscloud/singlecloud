package g53

import (
	"strings"

	"github.com/zdnscloud/g53/util"
)

type Question struct {
	Name  *Name
	Type  RRType
	Class RRClass
}

func QuestionFromWire(buf *util.InputBuffer) (*Question, error) {
	n, err := NameFromWire(buf, false)
	if err != nil {
		return nil, err
	}

	t, err := TypeFromWire(buf)
	if err != nil {
		return nil, err
	}

	cls, err := ClassFromWire(buf)
	if err != nil {
		return nil, err
	}

	return &Question{
		Name:  n,
		Type:  t,
		Class: cls,
	}, nil
}

func (q *Question) Rend(r *MsgRender) {
	q.Name.Rend(r)
	q.Type.Rend(r)
	q.Class.Rend(r)
}

func (q *Question) ToWire(buf *util.OutputBuffer) {
	q.Name.ToWire(buf)
	q.Type.ToWire(buf)
	q.Class.ToWire(buf)
}

func (q *Question) String() string {
	return strings.Join([]string{q.Name.String(false), q.Class.String(), q.Type.String()}, " ")
}

func (q *Question) Equals(o *Question) bool {
	return q.Name.Equals(o.Name) &&
		q.Type == o.Type &&
		q.Class == o.Class
}
