package g53

import (
	"errors"
	"math"
	"regexp"
	"strings"

	"github.com/zdnscloud/g53/util"
)

type SRV struct {
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   *Name
}

func (srv *SRV) Rend(r *MsgRender) {
	rendField(RDF_C_UINT16, srv.Priority, r)
	rendField(RDF_C_UINT16, srv.Weight, r)
	rendField(RDF_C_UINT16, srv.Port, r)
	rendField(RDF_C_NAME_UNCOMPRESS, srv.Target, r)
}

func (srv *SRV) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_UINT16, srv.Priority, buf)
	fieldToWire(RDF_C_UINT16, srv.Weight, buf)
	fieldToWire(RDF_C_UINT16, srv.Port, buf)
	fieldToWire(RDF_C_NAME, srv.Target, buf)
}

func (srv *SRV) Compare(other Rdata) int {
	otherSRV := other.(*SRV)
	order := fieldCompare(RDF_C_UINT16, srv.Priority, otherSRV.Priority)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT16, srv.Weight, otherSRV.Weight)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT16, srv.Port, otherSRV.Port)
	if order != 0 {
		return order
	}

	return fieldCompare(RDF_C_NAME, srv.Target, otherSRV.Target)
}

func (srv *SRV) String() string {
	var ss []string
	ss = append(ss, fieldToString(RDF_D_INT, srv.Priority))
	ss = append(ss, fieldToString(RDF_D_INT, srv.Weight))
	ss = append(ss, fieldToString(RDF_D_INT, srv.Port))
	ss = append(ss, fieldToString(RDF_D_NAME, srv.Target))
	return strings.Join(ss, " ")
}

func SRVFromWire(buf *util.InputBuffer, ll uint16) (*SRV, error) {
	p, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}

	w, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}

	port, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}

	t, ll, err := fieldFromWire(RDF_C_NAME, buf, ll)
	if err != nil {
		return nil, err
	}

	if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	}

	return &SRV{p.(uint16), w.(uint16), port.(uint16), t.(*Name)}, nil
}

var srvRdataTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)

func SRVFromString(s string) (*SRV, error) {
	fields := srvRdataTemplate.FindStringSubmatch(s)
	if len(fields) != 5 {
		return nil, errors.New("short of fields for srv")
	}
	fields = fields[1:]

	p, err := fieldFromString(RDF_D_INT, fields[0])
	if err != nil {
		return nil, err
	}
	priority, _ := p.(int)
	if priority > math.MaxUint16 {
		return nil, ErrOutOfRange
	}

	w, err := fieldFromString(RDF_D_INT, fields[1])
	if err != nil {
		return nil, err
	}
	weight, _ := w.(int)
	if weight > math.MaxUint16 {
		return nil, ErrOutOfRange
	}

	p, err = fieldFromString(RDF_D_INT, fields[2])
	if err != nil {
		return nil, err
	}
	port, _ := p.(int)
	if port > math.MaxUint16 {
		return nil, ErrOutOfRange
	}

	t, err := fieldFromString(RDF_D_NAME, fields[3])
	if err != nil {
		return nil, err
	}
	target, _ := t.(*Name)

	return &SRV{uint16(priority), uint16(weight), uint16(port), target}, nil
}
