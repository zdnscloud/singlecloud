package g53

import (
	"bytes"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/zdnscloud/g53/util"
)

type NSEC3 struct {
	Algorithm  uint8
	Flags      uint8
	Iterations uint16
	SaltLength uint8
	Salt       string
	HashLength uint8
	NextHash   string
	Types      []RRType
}

func (nsec3 *NSEC3) String() string {
	var buf bytes.Buffer
	buf.WriteString(fieldToString(RDF_D_INT, nsec3.Algorithm))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_INT, nsec3.Flags))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_INT, nsec3.Iterations))
	buf.WriteString(" ")
	if nsec3.SaltLength == 0 {
		nsec3.Salt = "-"
	}
	buf.WriteString(fieldToString(RDF_D_STR, nsec3.Salt))
	buf.WriteString(" ")
	buf.WriteString(fieldToString(RDF_D_STR, nsec3.NextHash))
	for _, typ := range nsec3.Types {
		buf.WriteString(" ")
		buf.WriteString(fieldToString(RDF_D_STR, typ.String()))
	}
	return buf.String()
}

func (nsec3 *NSEC3) Compare(other Rdata) int {
	otherNSEC3 := other.(*NSEC3)
	order := fieldCompare(RDF_C_UINT8, nsec3.Algorithm, otherNSEC3.Algorithm)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT8, nsec3.Flags, otherNSEC3.Flags)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT16, nsec3.Iterations, otherNSEC3.Iterations)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT8, nsec3.SaltLength, otherNSEC3.SaltLength)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_BYTE_BINARY, []byte(nsec3.Salt), []byte(otherNSEC3.Salt))
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_UINT8, nsec3.HashLength, otherNSEC3.HashLength)
	if order != 0 {
		return order
	}

	order = fieldCompare(RDF_C_BYTE_BINARY, []byte(nsec3.NextHash), []byte(otherNSEC3.NextHash))
	if order != 0 {
		return order
	}

	for i, typ := range nsec3.Types {
		if order := fieldCompare(RDF_C_UINT16, uint16(typ), uint16(otherNSEC3.Types[i])); order != 0 {
			return order
		}
	}

	return 0
}

func (nsec3 *NSEC3) Rend(r *MsgRender) {
	rendField(RDF_C_UINT8, nsec3.Algorithm, r)
	rendField(RDF_C_UINT8, nsec3.Flags, r)
	rendField(RDF_C_UINT16, nsec3.Iterations, r)
	rendField(RDF_C_UINT8, nsec3.SaltLength, r)
	rendField(RDF_C_BINARY, encodeStringToHex(nsec3.Salt), r)
	rendField(RDF_C_UINT8, nsec3.HashLength, r)
	rendField(RDF_C_BINARY, encodeNSEC3NextHash([]byte(nsec3.NextHash)), r)
	rendField(RDF_C_BINARY, encodeNSEC3Bytes(nsec3.Types), r)
}

func (nsec3 *NSEC3) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_UINT8, nsec3.Algorithm, buf)
	fieldToWire(RDF_C_UINT8, nsec3.Flags, buf)
	fieldToWire(RDF_C_UINT16, nsec3.Iterations, buf)
	fieldToWire(RDF_C_UINT8, nsec3.SaltLength, buf)
	fieldToWire(RDF_C_BINARY, encodeStringToHex(nsec3.Salt), buf)
	fieldToWire(RDF_C_UINT8, nsec3.HashLength, buf)
	fieldToWire(RDF_C_BINARY, encodeNSEC3NextHash([]byte(nsec3.NextHash)), buf)
	fieldToWire(RDF_C_BINARY, encodeNSEC3Bytes(nsec3.Types), buf)
}

func encodeStringToHex(saltStr string) []byte {
	salt, _ := hex.DecodeString(saltStr)
	return salt
}

func encodeNSEC3NextHash(nextHash []byte) []byte {
	buflen := base32.HexEncoding.DecodedLen(len(nextHash))
	buf := make([]byte, buflen)
	n, _ := base32.HexEncoding.Decode(buf, nextHash)
	buf = buf[:n]
	return buf
}

func encodeNSEC3Bytes(nsec3Types []RRType) []byte {
	types := make([]byte, (2+32)*len(nsec3Types))
	var lastwindow, lastlength uint16
	offset := 0
	for _, typ := range nsec3Types {
		window := uint16(typ) / 256
		length := (uint16(typ)-window*256)/8 + 1
		if window > lastwindow && lastlength != 0 {
			offset += int(lastlength) + 2
			lastlength = 0
		}

		types[offset] = byte(window)
		types[offset+1] = byte(length)
		types[offset+1+int(length)] |= byte(1 << (7 - (uint16(typ) % 8)))
		lastwindow, lastlength = window, length
	}

	return types[:offset+int(lastlength)+2]
}

func NSEC3FromWire(buf *util.InputBuffer, ll uint16) (*NSEC3, error) {
	algorithm, ll, err := fieldFromWire(RDF_C_UINT8, buf, ll)
	if err != nil {
		return nil, err
	}

	flags, ll, err := fieldFromWire(RDF_C_UINT8, buf, ll)
	if err != nil {
		return nil, err
	}

	iterations, ll, err := fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}

	saltLen, ll, err := fieldFromWire(RDF_C_UINT8, buf, ll)
	if err != nil {
		return nil, err
	}

	salt, _, err := fieldFromWire(RDF_C_BINARY, buf, uint16(saltLen.(uint8)))
	if err != nil {
		return nil, err
	} else {
		ll -= uint16(saltLen.(uint8))
	}

	hashLen, ll, err := fieldFromWire(RDF_C_UINT8, buf, ll)
	if err != nil {
		return nil, err
	}

	nextHash, _, err := fieldFromWire(RDF_C_BINARY, buf, uint16(hashLen.(uint8)))
	if err != nil {
		return nil, err
	} else {
		ll -= uint16(hashLen.(uint8))
	}

	nsec3Types, ll, err := fieldFromWire(RDF_C_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}

	if ll != 0 {
		return nil, fmt.Errorf("extra data in rdata part")
	}

	types, err := decodeNSEC3Types(nsec3Types.([]byte))
	if err != nil {
		return nil, err
	}

	return &NSEC3{
		Algorithm:  algorithm.(uint8),
		Flags:      flags.(uint8),
		Iterations: iterations.(uint16),
		SaltLength: saltLen.(uint8),
		Salt:       hex.EncodeToString(salt.([]uint8)),
		HashLength: hashLen.(uint8),
		NextHash:   base32.HexEncoding.EncodeToString(nextHash.([]uint8)),
		Types:      types,
	}, nil
}

func decodeNSEC3Types(msg []byte) ([]RRType, error) {
	var nsec3Types []RRType
	length, window, lastwindow := 0, 0, -1
	offset := 0
	for offset < len(msg) {
		if offset+2 > len(msg) {
			return nil, fmt.Errorf("overflow unpacking NSEC3 types")
		}

		window = int(msg[offset])
		length = int(msg[offset+1])
		offset += 2
		if window <= lastwindow {
			return nil, fmt.Errorf("out of order NSEC3 types block")
		}

		if length == 0 {
			return nil, fmt.Errorf("empty NSEC3 Types")
		}

		if length > 32 {
			return nil, fmt.Errorf("NSEC3 Types longer than 32")
		}

		if offset+length > len(msg) {
			return nil, fmt.Errorf("overflowing NSEC3 types")
		}

		for j := 0; j < length; j++ {
			b := msg[offset+j]
			base := window*256 + j*8
			for i := 7; i >= 0; i-- {
				if (b>>uint16(i))&0x01 != 0 {
					nsec3Types = append(nsec3Types, RRType(base+7-i))
				}
			}
		}
		offset += length
		lastwindow = window
	}

	return nsec3Types, nil
}

var nsec3RdataTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*?)\s*$`)
var nsec3TypesTemplate = regexp.MustCompile(`\s+`)

func NSEC3FromString(s string) (*NSEC3, error) {
	fields := nsec3RdataTemplate.FindStringSubmatch(s)
	if len(fields) != 9 {
		return nil, fmt.Errorf("short of fields for nsec3")
	}

	fields = fields[1:]
	algorithm, err := fieldFromString(RDF_D_INT, fields[0])
	if err != nil {
		return nil, err
	}

	flags, err := fieldFromString(RDF_D_INT, fields[1])
	if err != nil {
		return nil, err
	}

	iterations, err := fieldFromString(RDF_D_INT, fields[2])
	if err != nil {
		return nil, err
	}

	saltLen, err := fieldFromString(RDF_D_INT, fields[3])
	if err != nil {
		return nil, err
	}

	salt, err := fieldFromString(RDF_D_STR, fields[4])
	if err != nil {
		return nil, err
	}

	if len(salt.(string)) != saltLen.(int) {
		return nil, fmt.Errorf("nsec3 from string failed with salt_len %v not equal len of salt %v",
			saltLen.(int), len(salt.(string)))
	}

	hashLen, err := fieldFromString(RDF_D_INT, fields[5])
	if err != nil {
		return nil, err
	}

	nextHash, err := fieldFromString(RDF_D_STR, fields[6])
	if err != nil {
		return nil, err
	}

	if len(nextHash.(string)) != hashLen.(int) {
		return nil, fmt.Errorf("nsec3 from string failed with hash_len %v and len of next_hash_domain %v",
			hashLen.(int), len(nextHash.(string)))
	}

	var types []RRType
	for _, field := range nsec3TypesTemplate.Split(fields[7], -1) {
		typ, err := TypeFromString(field)
		if err != nil {
			return nil, err
		} else {
			types = append(types, typ)
		}
	}

	return &NSEC3{
		Algorithm:  uint8(algorithm.(int)),
		Flags:      uint8(flags.(int)),
		Iterations: uint16(iterations.(int)),
		SaltLength: uint8(saltLen.(int)),
		Salt:       salt.(string),
		HashLength: uint8(hashLen.(int)),
		NextHash:   nextHash.(string),
		Types:      types,
	}, nil
}
