package g53

import (
	"bytes"
	"errors"
	"regexp"

	"github.com/zdnscloud/g53/util"
)

type SOA struct {
	MName   *Name
	RName   *Name
	Serial  uint32
	Refresh uint32
	Retry   uint32
	Expire  uint32
	Minimum uint32
}

func (soa *SOA) Rend(r *MsgRender) {
	rendField(RDF_C_NAME, soa.MName, r)
	rendField(RDF_C_NAME, soa.RName, r)
	rendField(RDF_C_UINT32, soa.Serial, r)
	rendField(RDF_C_UINT32, soa.Refresh, r)
	rendField(RDF_C_UINT32, soa.Retry, r)
	rendField(RDF_C_UINT32, soa.Expire, r)
	rendField(RDF_C_UINT32, soa.Minimum, r)
}

func (soa *SOA) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_NAME, soa.MName, buf)
	fieldToWire(RDF_C_NAME, soa.RName, buf)
	fieldToWire(RDF_C_UINT32, soa.Serial, buf)
	fieldToWire(RDF_C_UINT32, soa.Refresh, buf)
	fieldToWire(RDF_C_UINT32, soa.Retry, buf)
	fieldToWire(RDF_C_UINT32, soa.Expire, buf)
	fieldToWire(RDF_C_UINT32, soa.Minimum, buf)
}

func (soa *SOA) Compare(other Rdata) int {
	return 0 //soa rrset should has one rr
}

func (soa *SOA) String() string {
	var buf bytes.Buffer
	buf.WriteString(fieldToString(RDF_D_NAME, soa.MName))
	buf.WriteByte(' ')
	buf.WriteString(fieldToString(RDF_D_NAME, soa.RName))
	buf.WriteByte(' ')
	buf.WriteString(fieldToString(RDF_D_INT, soa.Serial))
	buf.WriteByte(' ')
	buf.WriteString(fieldToString(RDF_D_INT, soa.Refresh))
	buf.WriteByte(' ')
	buf.WriteString(fieldToString(RDF_D_INT, soa.Retry))
	buf.WriteByte(' ')
	buf.WriteString(fieldToString(RDF_D_INT, soa.Expire))
	buf.WriteByte(' ')
	buf.WriteString(fieldToString(RDF_D_INT, soa.Minimum))
	buf.WriteByte(' ')

	return buf.String()
}

func SOAFromWire(buf *util.InputBuffer, ll uint16) (*SOA, error) {
	name, ll, err := fieldFromWire(RDF_C_NAME, buf, ll)
	if err != nil {
		return nil, err
	}
	mname, _ := name.(*Name)

	name, ll, err = fieldFromWire(RDF_C_NAME, buf, ll)
	if err != nil {
		return nil, err
	}
	rname, _ := name.(*Name)

	i, ll, err := fieldFromWire(RDF_C_UINT32, buf, ll)
	if err != nil {
		return nil, err
	}
	serial, _ := i.(uint32)

	i, ll, err = fieldFromWire(RDF_C_UINT32, buf, ll)
	if err != nil {
		return nil, err
	}
	refresh, _ := i.(uint32)

	i, ll, err = fieldFromWire(RDF_C_UINT32, buf, ll)
	if err != nil {
		return nil, err
	}
	retry, _ := i.(uint32)

	i, ll, err = fieldFromWire(RDF_C_UINT32, buf, ll)
	if err != nil {
		return nil, err
	}
	expire, _ := i.(uint32)

	i, ll, err = fieldFromWire(RDF_C_UINT32, buf, ll)
	if err != nil {
		return nil, err
	}
	minimum, _ := i.(uint32)

	if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	}

	return &SOA{mname, rname, serial, refresh, retry, expire, minimum}, nil
}

var soaRdataTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s*$`)

func SOAFromString(s string) (*SOA, error) {
	fields := soaRdataTemplate.FindStringSubmatch(s)
	if len(fields) != 8 {
		return nil, errors.New("short of fields for soa")
	}

	fields = fields[1:]
	name, err := fieldFromString(RDF_D_NAME, fields[0])
	if err != nil {
		return nil, err
	}
	mname, _ := name.(*Name)

	name, err = fieldFromString(RDF_D_NAME, fields[1])
	if err != nil {
		return nil, err
	}
	rname, _ := name.(*Name)

	i, err := fieldFromString(RDF_D_INT, fields[2])
	if err != nil {
		return nil, err
	}
	serial, _ := i.(int)

	i, err = fieldFromString(RDF_D_INT, fields[3])
	if err != nil {
		return nil, err
	}
	refresh, _ := i.(int)

	i, err = fieldFromString(RDF_D_INT, fields[4])
	if err != nil {
		return nil, err
	}
	retry, _ := i.(int)

	i, err = fieldFromString(RDF_D_INT, fields[5])
	if err != nil {
		return nil, err
	}
	expire, _ := i.(int)

	i, err = fieldFromString(RDF_D_INT, fields[6])
	if err != nil {
		return nil, err
	}
	minimum, _ := i.(int)

	return &SOA{mname, rname, uint32(serial), uint32(refresh), uint32(retry), uint32(expire), uint32(minimum)}, nil
}

const year68 = 1 << 31 // For RFC1982 (Serial Arithmetic)
func CompareSerial(serial1, serial2 uint32) int {
	if serial1 == serial2 {
		return 0
	}

	if serial1 < serial2 {
		if serial2-serial1 < year68 {
			return -1
		} else {
			return 1
		}
	} else {
		if serial1-serial2 < year68 {
			return 1
		} else {
			return -1
		}
	}
}
