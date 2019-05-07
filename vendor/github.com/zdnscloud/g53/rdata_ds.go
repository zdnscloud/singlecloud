package g53

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/zdnscloud/g53/util"
)

type DS struct {
	KeyTag     uint16
	Algorithm  uint8
	DigestType uint8
	Digest     string
}

func (ds *DS) String() string {
	var buf bytes.Buffer
	buf.WriteString(fieldToString(RDF_D_INT, ds.KeyTag))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_INT, ds.Algorithm))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_INT, ds.DigestType))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_STR, strings.ToUpper(ds.Digest)))
	return buf.String()
}

func (ds *DS) Compare(other Rdata) int {
	otherDS := other.(*DS)

	order := fieldCompare(RDF_C_UINT16, ds.KeyTag, otherDS.KeyTag)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT8, ds.Algorithm, otherDS.Algorithm)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT8, ds.DigestType, otherDS.DigestType)
	if order != 0 {
		return order
	}

	return fieldCompare(RDF_C_BYTE_BINARY, []byte(ds.Digest), []byte(otherDS.Digest))
}

func (ds *DS) Rend(r *MsgRender) {
	rendField(RDF_C_UINT16, ds.KeyTag, r)
	rendField(RDF_C_UINT8, ds.Algorithm, r)
	rendField(RDF_C_UINT8, ds.DigestType, r)
	rendField(RDF_C_BINARY, encodeStringToHex(ds.Digest), r)
}

func (ds *DS) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_UINT16, ds.KeyTag, buf)
	fieldToWire(RDF_C_UINT8, ds.Algorithm, buf)
	fieldToWire(RDF_C_UINT8, ds.DigestType, buf)
	fieldToWire(RDF_C_BINARY, encodeStringToHex(ds.Digest), buf)
}

func DSFromWire(buf *util.InputBuffer, ll uint16) (*DS, error) {
	keyTag, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}

	algorithm, ll, err := fieldFromWire(RDF_C_UINT8, buf, ll)
	if err != nil {
		return nil, err
	}

	typ, ll, err := fieldFromWire(RDF_C_UINT8, buf, ll)
	if err != nil {
		return nil, err
	}

	digest, ll, err := fieldFromWire(RDF_C_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}

	if ll != 0 {
		return nil, fmt.Errorf("extra data in rdata part")
	}

	return &DS{
		KeyTag:     keyTag.(uint16),
		Algorithm:  algorithm.(uint8),
		DigestType: typ.(uint8),
		Digest:     hex.EncodeToString(digest.([]uint8)),
	}, nil
}

var dsRdataTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(.*?)\s*$`)
var dsDigestTemplate = regexp.MustCompile(`\s+`)

func DSFromString(s string) (*DS, error) {
	fields := dsRdataTemplate.FindStringSubmatch(s)
	if len(fields) != 5 {
		return nil, fmt.Errorf("short of fields for ds")
	}

	fields = fields[1:]
	keyTag, err := fieldFromString(RDF_D_INT, fields[0])
	if err != nil {
		return nil, err
	}

	algorithm, err := fieldFromString(RDF_D_INT, fields[1])
	if err != nil {
		return nil, err
	}

	typ, err := fieldFromString(RDF_D_INT, fields[2])
	if err != nil {
		return nil, err
	}

	digest, err := fieldFromString(RDF_D_STR, fields[3])
	if err != nil {
		return nil, err
	}

	return &DS{
		KeyTag:     uint16(keyTag.(int)),
		Algorithm:  uint8(algorithm.(int)),
		DigestType: uint8(typ.(int)),
		Digest:     dsDigestTemplate.ReplaceAllString(digest.(string), ""),
	}, nil
}
