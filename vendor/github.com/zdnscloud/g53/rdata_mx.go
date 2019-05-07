package g53

import (
	"errors"
	"math"
	"regexp"
	"strings"

	"github.com/zdnscloud/g53/util"
)

type MX struct {
	Preference uint16
	Exchange   *Name
}

func (mx *MX) Rend(r *MsgRender) {
	rendField(RDF_C_UINT16, mx.Preference, r)
	rendField(RDF_C_NAME, mx.Exchange, r)
}

func (mx *MX) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_UINT16, mx.Preference, buf)
	fieldToWire(RDF_C_NAME, mx.Exchange, buf)
}

func (mx *MX) Compare(other Rdata) int {
	otherMX := other.(*MX)
	order := fieldCompare(RDF_C_UINT16, mx.Preference, otherMX.Preference)
	if order != 0 {
		return order
	}

	return fieldCompare(RDF_C_NAME, mx.Exchange, otherMX.Exchange)
}

func (mx *MX) String() string {
	return strings.Join([]string{
		fieldToString(RDF_D_INT, mx.Preference),
		fieldToString(RDF_D_NAME, mx.Exchange)}, " ")
}

func MXFromWire(buf *util.InputBuffer, ll uint16) (*MX, error) {
	f, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	preference, _ := f.(uint16)

	f, ll, err = fieldFromWire(RDF_C_NAME, buf, ll)
	if err != nil {
		return nil, err
	}
	exchange, _ := f.(*Name)

	if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	}

	return &MX{preference, exchange}, nil
}

var mxRdataTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s*$`)

func MXFromString(s string) (*MX, error) {
	fields := mxRdataTemplate.FindStringSubmatch(s)
	if len(fields) != 3 {
		return nil, errors.New("fields count for mx isn't 2")
	}

	fields = fields[1:]
	f, err := fieldFromString(RDF_D_INT, fields[0])
	if err != nil {
		return nil, err
	}
	preference, _ := f.(int)
	if preference > math.MaxUint16 {
		return nil, ErrOutOfRange
	}

	f, err = fieldFromString(RDF_D_NAME, fields[1])
	if err != nil {
		return nil, err
	}
	exchange, _ := f.(*Name)
	return &MX{uint16(preference), exchange}, nil
}
