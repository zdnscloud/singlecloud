package g53

import (
	"errors"
	"github.com/zdnscloud/g53/util"
)

type DName struct {
	Target *Name
}

func (c *DName) Rend(r *MsgRender) {
	rendField(RDF_C_NAME, c.Target, r)
}

func (c *DName) ToWire(buffer *util.OutputBuffer) {
	fieldToWire(RDF_C_NAME, c.Target, buffer)
}

func (c *DName) Compare(other Rdata) int {
	return fieldCompare(RDF_C_NAME, c.Target, other.(*DName).Target)
}

func (c *DName) String() string {
	return fieldToString(RDF_D_NAME, c.Target)
}

func DNameFromWire(buffer *util.InputBuffer, ll uint16) (*DName, error) {
	n, ll, err := fieldFromWire(RDF_C_NAME, buffer, ll)

	if err != nil {
		return nil, err
	} else if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	} else {
		name, _ := n.(*Name)
		return &DName{name}, nil
	}
}

func DNameFromString(s string) (*DName, error) {
	n, err := fieldFromString(RDF_D_NAME, s)
	if err == nil {
		name, _ := n.(*Name)
		return &DName{name}, nil
	} else {
		return nil, err
	}
}
