package g53

import (
	"errors"

	"github.com/zdnscloud/g53/util"
)

type SPF struct {
	Data []string
}

func (spf *SPF) Rend(r *MsgRender) {
	rendField(RDF_C_TXT, spf.Data, r)
}

func (spf *SPF) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_TXT, spf.Data, buf)
}

func (spf *SPF) Compare(other Rdata) int {
	return fieldCompare(RDF_C_TXT, spf.Data, other.(*SPF).Data)
}

func (spf *SPF) String() string {
	return fieldToString(RDF_D_TXT, spf.Data)
}

func SPFFromWire(buf *util.InputBuffer, ll uint16) (*SPF, error) {
	f, ll, err := fieldFromWire(RDF_C_TXT, buf, ll)
	if err != nil {
		return nil, err
	} else if ll != 0 {
		return nil, errors.New("extra data in rdata part when parse spf")
	} else {
		data, _ := f.([]string)
		return &SPF{data}, nil
	}
}

func SPFFromString(s string) (*SPF, error) {
	f, err := fieldFromString(RDF_D_TXT, s)
	if err != nil {
		return nil, err
	} else {
		return &SPF{f.([]string)}, nil
	}
}
