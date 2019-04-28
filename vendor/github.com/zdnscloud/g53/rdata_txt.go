package g53

import (
	"errors"

	"github.com/zdnscloud/g53/util"
)

type Txt struct {
	Data []string
}

func (txt *Txt) Rend(r *MsgRender) {
	rendField(RDF_C_TXT, txt.Data, r)
}

func (txt *Txt) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_TXT, txt.Data, buf)
}

func (txt *Txt) Compare(other Rdata) int {
	return fieldCompare(RDF_C_TXT, txt.Data, other.(*Txt).Data)
}

func (txt *Txt) String() string {
	return fieldToString(RDF_D_TXT, txt.Data)
}

func TxtFromWire(buf *util.InputBuffer, ll uint16) (*Txt, error) {
	f, ll, err := fieldFromWire(RDF_C_TXT, buf, ll)
	if err != nil {
		return nil, err
	} else if ll != 0 {
		return nil, errors.New("extra data in rdata part when parse txt")
	} else {
		data, _ := f.([]string)
		return &Txt{data}, nil
	}
}

func TxtFromString(s string) (*Txt, error) {
	f, err := fieldFromString(RDF_D_TXT, s)
	if err != nil {
		return nil, err
	} else {
		return &Txt{f.([]string)}, nil
	}
}
