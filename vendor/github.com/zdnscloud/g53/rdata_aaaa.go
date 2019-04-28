package g53

import (
	"errors"
	"net"

	"github.com/zdnscloud/g53/util"
)

type AAAA struct {
	Host net.IP
}

func (aaaa *AAAA) Rend(r *MsgRender) {
	rendField(RDF_C_IPV6, aaaa.Host, r)
}

func (aaaa *AAAA) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_IPV6, aaaa.Host, buf)
}

func (aaaa *AAAA) Compare(other Rdata) int {
	return fieldCompare(RDF_C_IPV6, aaaa.Host, other.(*AAAA).Host)
}

func (aaaa *AAAA) String() string {
	return fieldToString(RDF_D_IP, aaaa.Host)
}

func AAAAFromWire(buf *util.InputBuffer, ll uint16) (*AAAA, error) {
	f, ll, err := fieldFromWire(RDF_C_IPV6, buf, ll)
	if err != nil {
		return nil, err
	} else if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	} else {
		host, _ := f.(net.IP)
		return &AAAA{host.To16()}, nil
	}
}

func AAAAFromString(s string) (*AAAA, error) {
	f, err := fieldFromString(RDF_D_IP, s)
	if err == nil {
		host, _ := f.(net.IP)
		return &AAAA{host}, nil
	} else {
		return nil, err
	}
}
