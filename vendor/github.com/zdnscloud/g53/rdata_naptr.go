package g53

import (
	"bytes"
	"errors"
	"math"
	"regexp"
	"strings"

	"github.com/zdnscloud/g53/util"
)

type NAPTR struct {
	Order       uint16
	Preference  uint16
	Flags       string
	Services    string
	Regexp      string
	Replacement *Name
}

func (naptr *NAPTR) Rend(r *MsgRender) {
	rendField(RDF_C_UINT16, naptr.Order, r)
	rendField(RDF_C_UINT16, naptr.Preference, r)
	rendField(RDF_C_BYTE_BINARY, []byte(naptr.Flags), r)
	rendField(RDF_C_BYTE_BINARY, []byte(naptr.Services), r)
	rendField(RDF_C_BYTE_BINARY, []byte(naptr.Regexp), r)
	rendField(RDF_C_NAME, naptr.Replacement, r)
}

func (naptr *NAPTR) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_UINT16, naptr.Order, buf)
	fieldToWire(RDF_C_UINT16, naptr.Preference, buf)
	fieldToWire(RDF_C_BYTE_BINARY, []byte(naptr.Flags), buf)
	fieldToWire(RDF_C_BYTE_BINARY, []byte(naptr.Services), buf)
	fieldToWire(RDF_C_BYTE_BINARY, []byte(naptr.Regexp), buf)
	fieldToWire(RDF_C_NAME, naptr.Replacement, buf)
}

func (naptr *NAPTR) Compare(other Rdata) int {
	otherNAPTR := other.(*NAPTR)
	order := fieldCompare(RDF_C_UINT16, naptr.Order, otherNAPTR.Order)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT16, naptr.Preference, otherNAPTR.Preference)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_BYTE_BINARY, []byte(naptr.Flags), []byte(otherNAPTR.Flags))
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_BYTE_BINARY, []byte(naptr.Services), []byte(otherNAPTR.Services))
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_BYTE_BINARY, []byte(naptr.Regexp), []byte(otherNAPTR.Regexp))
	if order != 0 {
		return order
	}

	return fieldCompare(RDF_C_NAME, naptr.Replacement, otherNAPTR.Replacement)
}

func (naptr *NAPTR) String() string {
	var buf bytes.Buffer
	buf.WriteString(fieldToString(RDF_D_INT, naptr.Order))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_INT, naptr.Preference))
	buf.WriteString(" ")
	buf.WriteString(strings.Join([]string{"\"", fieldToString(RDF_D_STR, naptr.Flags), "\""}, ""))
	buf.WriteString(" ")
	buf.WriteString(strings.Join([]string{"\"", fieldToString(RDF_D_STR, naptr.Services), "\""}, ""))
	buf.WriteString(" ")
	buf.WriteString(strings.Join([]string{"\"", fieldToString(RDF_D_STR, naptr.Regexp), "\""}, ""))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_NAME, naptr.Replacement))
	return buf.String()
}

func NAPTRFromWire(buf *util.InputBuffer, ll uint16) (*NAPTR, error) {
	o, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	order, _ := o.(uint16)

	p, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	preference, _ := p.(uint16)

	f, ll, err := fieldFromWire(RDF_C_BYTE_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}
	f_, _ := f.([]uint8)
	flags := string(f_)

	s, ll, err := fieldFromWire(RDF_C_BYTE_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}
	s_, _ := s.([]uint8)
	service := string(s_)

	r, ll, err := fieldFromWire(RDF_C_BYTE_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}
	r_, _ := r.([]uint8)
	regex := string(r_)

	n, ll, err := fieldFromWire(RDF_C_NAME, buf, ll)
	if err != nil {
		return nil, err
	}
	replacement, _ := n.(*Name)

	if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	}

	return &NAPTR{order, preference, flags, service, regex, replacement}, nil
}

var naptrRdataTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)

func NAPTRFromString(s string) (*NAPTR, error) {
	fields := naptrRdataTemplate.FindStringSubmatch(s)
	if len(fields) != 7 {
		return nil, errors.New("short of fields for naptr")
	}

	fields = fields[1:]
	o, err := fieldFromString(RDF_D_INT, fields[0])
	if err != nil {
		return nil, err
	}
	order, _ := o.(int)
	if order > math.MaxUint16 {
		return nil, ErrOutOfRange
	}

	p, err := fieldFromString(RDF_D_INT, fields[1])
	if err != nil {
		return nil, err
	}
	preference, _ := p.(int)
	if preference > math.MaxUint16 {
		return nil, ErrOutOfRange
	}

	f, err := fieldFromString(RDF_D_STR, fields[2])
	if err != nil {
		return nil, err
	}
	flags, _ := f.(string)

	se, err := fieldFromString(RDF_D_STR, fields[3])
	if err != nil {
		return nil, err
	}
	service, _ := se.(string)

	r, err := fieldFromString(RDF_D_STR, fields[4])
	if err != nil {
		return nil, err
	}
	regex, _ := r.(string)

	n, err := fieldFromString(RDF_D_NAME, fields[5])
	if err != nil {
		return nil, err
	}
	replacement, _ := n.(*Name)

	return &NAPTR{uint16(order), uint16(preference), flags, service, regex, replacement}, nil
}
