package g53

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/zdnscloud/g53/util"
)

var (
	ErrStringIsTooLong       = errors.New("character string is too long")
	ErrInvalidIPAddr         = errors.New("invalid ip address")
	ErrQuoteInTxtIsNotInPair = errors.New("quote in text record isn't in pair")
	ErrDataIsTooShort        = errors.New("raw data isn't long enough")
	ErrOutOfRange            = errors.New("data out of range")
	ErrInvalidTXT            = errors.New("txt record is not valid")
)

type RDFCodingType uint8
type RDFDisplayType uint8

const (
	RDF_C_NAME RDFCodingType = iota
	RDF_C_NAME_UNCOMPRESS
	RDF_C_UINT8
	RDF_C_UINT16
	RDF_C_UINT32
	RDF_C_IPV4
	RDF_C_IPV6
	RDF_C_BINARY
	RDF_C_BYTE_BINARY //<character-string>
	RDF_C_TXT
)

const (
	RDF_D_NAME RDFDisplayType = iota
	RDF_D_INT
	RDF_D_IP
	RDF_D_TXT
	RDF_D_HEX
	RDF_D_B32
	RDF_D_B64
	RDF_D_STR
)

func fieldFromWire(ct RDFCodingType, buf *util.InputBuffer, ll uint16) (interface{}, uint16, error) {
	switch ct {
	case RDF_C_NAME, RDF_C_NAME_UNCOMPRESS:
		pos := buf.Position()
		n, err := NameFromWire(buf, true)
		if err != nil {
			return nil, ll, err
		} else {
			namelen := uint16(buf.Position() - pos)
			if ll < namelen {
				return nil, ll, ErrDataIsTooShort
			} else {
				return n, ll - namelen, nil
			}
		}

	case RDF_C_UINT8:
		d, err := buf.ReadUint8()
		if err != nil {
			return nil, ll, err
		} else if ll < 1 {
			return nil, ll, ErrDataIsTooShort
		} else {
			return d, ll - 1, nil
		}

	case RDF_C_UINT16:
		d, err := buf.ReadUint16()
		if err != nil {
			return nil, ll, err
		} else if ll < 2 {
			return nil, ll, ErrDataIsTooShort
		} else {
			return d, ll - 2, nil
		}

	case RDF_C_UINT32:
		d, err := buf.ReadUint32()
		if err != nil {
			return nil, ll, err
		} else if ll < 4 {
			return nil, ll, ErrDataIsTooShort
		} else {
			return d, ll - 4, nil
		}

	case RDF_C_IPV4:
		d, err := buf.ReadBytes(4)
		if err != nil {
			return nil, ll, err
		} else if ll < 4 {
			return nil, ll, ErrDataIsTooShort
		} else {
			clone := make([]byte, 4)
			copy(clone, d)
			return net.IP(clone), ll - 4, nil
		}

	case RDF_C_IPV6:
		d, err := buf.ReadBytes(16)
		if err != nil {
			return nil, ll, err
		} else if ll < 16 {
			return nil, ll, ErrDataIsTooShort
		} else {
			clone := make([]byte, 16)
			copy(clone, d)
			return net.IP(clone), ll - 16, nil
		}

	case RDF_C_TXT:
		var ss []string
		var d interface{}
		var err error
		for ll > 0 {
			d, ll, err = fieldFromWire(RDF_C_BYTE_BINARY, buf, ll)
			if err != nil {
				return nil, ll, err
			}
			bs, _ := d.([]uint8)
			ss = append(ss, string(bs))
		}
		return ss, 0, nil

	case RDF_C_BINARY:
		d, err := buf.ReadBytes(uint(ll))
		if err != nil {
			return nil, ll, err
		}

		clone := make([]byte, ll)
		copy(clone, d)
		return clone, 0, nil

	case RDF_C_BYTE_BINARY:
		l, err := buf.ReadUint8()
		if err != nil {
			return nil, ll, err
		}
		if ll < 1 {
			return nil, ll, ErrDataIsTooShort
		}
		ll -= 1
		if uint16(l) > ll {
			return nil, ll, ErrStringIsTooLong
		}
		d, err := buf.ReadBytes(uint(l))
		if err != nil {
			return nil, ll, err
		}
		if ll < uint16(l) {
			return nil, ll, ErrDataIsTooShort
		}

		clone := make([]byte, l)
		copy(clone, d)
		return clone, ll - uint16(l), nil

	default:
		panic("unknown rdata file type")
	}
}

func rendField(ct RDFCodingType, data interface{}, render *MsgRender) {
	switch ct {
	case RDF_C_NAME:
		n, _ := data.(*Name)
		n.Rend(render)

	case RDF_C_NAME_UNCOMPRESS:
		n, _ := data.(*Name)
		render.WriteName(n, false)

	case RDF_C_UINT8:
		d, _ := data.(uint8)
		render.WriteUint8(d)

	case RDF_C_UINT16:
		d, _ := data.(uint16)
		render.WriteUint16(d)

	case RDF_C_UINT32:
		d, _ := data.(uint32)
		render.WriteUint32(d)

	case RDF_C_IPV4, RDF_C_IPV6:
		d, _ := data.(net.IP)
		render.WriteData([]uint8(d))

	case RDF_C_BINARY:
		d, _ := data.([]uint8)
		render.WriteData(d)

	case RDF_C_TXT:
		ds, _ := data.([]string)
		for _, d := range ds {
			rendField(RDF_C_BYTE_BINARY, []uint8(d), render)
		}

	case RDF_C_BYTE_BINARY:
		d, _ := data.([]uint8)
		render.WriteUint8(uint8(len(d)))
		render.WriteData(d)
	}
}

func fieldToWire(ct RDFCodingType, data interface{}, buf *util.OutputBuffer) {
	switch ct {
	case RDF_C_NAME, RDF_C_NAME_UNCOMPRESS:
		n, _ := data.(*Name)
		n.ToWire(buf)

	case RDF_C_UINT8:
		d, _ := data.(uint8)
		buf.WriteUint8(d)

	case RDF_C_UINT16:
		d, _ := data.(uint16)
		buf.WriteUint16(d)

	case RDF_C_UINT32:
		d, _ := data.(uint32)
		buf.WriteUint32(d)

	case RDF_C_IPV4, RDF_C_IPV6:
		ip, _ := data.(net.IP)
		buf.WriteData([]uint8(ip))

	case RDF_C_BINARY:
		d, _ := data.([]uint8)
		buf.WriteData(d)

	case RDF_C_TXT:
		ds, _ := data.([]string)
		for _, d := range ds {
			fieldToWire(RDF_C_BYTE_BINARY, d, buf)
		}

	case RDF_C_BYTE_BINARY:
		d, _ := data.([]uint8)
		buf.WriteUint8(uint8(len(d)))
		buf.WriteData(d)
	}
}

func fieldCompare(ct RDFCodingType, data1 interface{}, data2 interface{}) int {
	switch ct {
	case RDF_C_NAME, RDF_C_NAME_UNCOMPRESS:
		n1, _ := data1.(*Name)
		n2, _ := data2.(*Name)
		return n1.Compare(n2, false).Order

	case RDF_C_UINT8:
		d1, _ := data1.(uint8)
		d2, _ := data2.(uint8)
		return int(d1) - int(d2)

	case RDF_C_UINT16:
		d1, _ := data1.(uint16)
		d2, _ := data2.(uint16)
		return int(d1) - int(d2)

	case RDF_C_UINT32:
		d1, _ := data1.(uint32)
		d2, _ := data2.(uint32)
		if d1 == d2 {
			return 0
		} else if d1 < d2 {
			return -1
		} else {
			return 1
		}

	case RDF_C_IPV4, RDF_C_IPV6:
		ip1, _ := data1.(net.IP)
		ip2, _ := data2.(net.IP)
		return bytes.Compare([]byte(ip1), []byte(ip2))

	case RDF_C_BINARY:
		d1, _ := data1.([]byte)
		d2, _ := data2.([]byte)
		return bytes.Compare(d1, d2)

	case RDF_C_TXT:
		ds1, _ := data1.([]string)
		ds2, _ := data2.([]string)
		return util.StringSliceCompare(ds1, ds2, true)

	case RDF_C_BYTE_BINARY:
		d1, _ := data1.([]byte)
		d2, _ := data2.([]byte)
		return bytes.Compare(d1, d2)
	}

	panic("unknown rr type")
	return 0
}

func fieldFromString(dt RDFDisplayType, s string) (interface{}, error) {
	switch dt {
	case RDF_D_NAME:
		n, err := NameFromString(s)
		if err != nil {
			return nil, err
		} else {
			return n, nil
		}

	case RDF_D_INT:
		d, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		} else {
			return d, nil
		}

	case RDF_D_IP:
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, ErrInvalidIPAddr
		} else {
			return ip, nil
		}

	case RDF_D_TXT:
		return txtStringParse(s)

	case RDF_D_HEX:
		d, err := util.HexStrToBytes(s)
		if err != nil {
			return nil, err
		} else {
			return d, nil
		}

	case RDF_D_B32:
		d, err := base32.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		} else {
			return []uint8(d), nil
		}

	case RDF_D_B64:
		d, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		} else {
			return []uint8(d), nil
		}

	case RDF_D_STR:
		return strings.Trim(s, "\""), nil

	default:
		panic("unknown display type")
	}
}

func fieldToString(dt RDFDisplayType, d interface{}) string {
	switch dt {
	case RDF_D_NAME:
		n, _ := d.(*Name)
		return n.String(false)

	case RDF_D_INT:
		return fmt.Sprintf("%v", d)

	case RDF_D_IP:
		ip, _ := d.(net.IP)
		return ip.String()

	case RDF_D_TXT:
		ss, _ := d.([]string)
		labels := []string{}
		for _, label := range ss {
			labels = append(labels, "\""+label+"\"")
		}
		return strings.Join(labels, " ")

	case RDF_D_HEX:
		bs, _ := d.([]uint8)
		s := ""
		for _, b := range bs {
			s += fmt.Sprintf("%x", b)
		}
		return s

	case RDF_D_B32:
		bs, _ := d.([]uint8)
		return base32.StdEncoding.EncodeToString([]byte(bs))

	case RDF_D_B64:
		bs, _ := d.([]uint8)
		return base64.StdEncoding.EncodeToString([]byte(bs))

	case RDF_D_STR:
		s, _ := d.(string)
		return s

	default:
		panic("unknown display type")
	}
}

//txt record rdata should be put into qoute
//it could include multi segment
var spaceReg = regexp.MustCompile(`\s+`)

func txtStringParse(txt string) ([]string, error) {
	s := strings.TrimSpace(txt)
	//add quote
	if strings.HasPrefix(s, "\"") == false {
		s = strings.Replace(s, "\"", "\\\"", -1) //only hanle one level embed
		s = "\"" + spaceReg.ReplaceAllString(s, "\" \"") + "\""
	}

	strs := []string{}
	inQuote := false
	startEscape := false
	lastPos := 0
	for i, c := range s {
		if c == '\\' {
			startEscape = true
		} else {
			if c == '"' && startEscape == false {
				if inQuote {
					strs = append(strs, s[lastPos:i])
					inQuote = false
				} else {
					inQuote = true
					lastPos = i + 1
				}
			}
			startEscape = false
		}
	}

	if inQuote {
		return nil, ErrQuoteInTxtIsNotInPair
	} else if len(strs) == 0 {
		return nil, ErrInvalidTXT
	} else {
		return strs, nil
	}
}
