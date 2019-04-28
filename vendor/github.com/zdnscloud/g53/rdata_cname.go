package g53

import (
	"errors"
	"github.com/zdnscloud/g53/util"
)

type CName struct {
	Name *Name
}

func (c *CName) Rend(r *MsgRender) {
	rendField(RDF_C_NAME, c.Name, r)
}

func (c *CName) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_NAME, c.Name, buf)
}

func (c *CName) String() string {
	return fieldToString(RDF_D_NAME, c.Name)
}

func (c *CName) Compare(other Rdata) int {
	return 0 //there should one rr in cname rrset
}

func CNameFromWire(buf *util.InputBuffer, ll uint16) (*CName, error) {
	n, ll, err := fieldFromWire(RDF_C_NAME, buf, ll)

	if err != nil {
		return nil, err
	} else if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	} else {
		name, _ := n.(*Name)
		return &CName{name}, nil
	}
}

func CNameFromString(s string) (*CName, error) {
	n, err := fieldFromString(RDF_D_NAME, s)
	if err == nil {
		name, _ := n.(*Name)
		return &CName{name}, nil
	} else {
		return nil, err
	}
}
