package g53

import (
	"fmt"

	"github.com/zdnscloud/g53/util"
)

type Rdata interface {
	Rend(r *MsgRender)
	ToWire(buf *util.OutputBuffer)
	Compare(Rdata) int
	String() string
}

func RdataFromWire(t RRType, buf *util.InputBuffer) (Rdata, error) {
	rdlen, err := buf.ReadUint16()
	if err != nil {
		return nil, err
	}

	if rdlen == 0 {
		return nil, nil
	}

	switch t {
	case RR_A:
		return AFromWire(buf, rdlen)
	case RR_AAAA:
		return AAAAFromWire(buf, rdlen)
	case RR_CNAME:
		return CNameFromWire(buf, rdlen)
	case RR_SOA:
		return SOAFromWire(buf, rdlen)
	case RR_NS:
		return NSFromWire(buf, rdlen)
	case RR_OPT:
		return OPTFromWire(buf, rdlen)
	case RR_PTR:
		return PTRFromWire(buf, rdlen)
	case RR_SRV:
		return SRVFromWire(buf, rdlen)
	case RR_NAPTR:
		return NAPTRFromWire(buf, rdlen)
	case RR_DNAME:
		return DNameFromWire(buf, rdlen)
	case RR_RRSIG:
		return RRSigFromWire(buf, rdlen)
	case RR_MX:
		return MXFromWire(buf, rdlen)
	case RR_TXT:
		return TxtFromWire(buf, rdlen)
	case RR_RP:
		return RPFromWire(buf, rdlen)
	case RR_SPF:
		return SPFFromWire(buf, rdlen)
	case RR_TSIG:
		return TSIGFromWire(buf, rdlen)
	case RR_NSEC3:
		return NSEC3FromWire(buf, rdlen)
	case RR_DS:
		return DSFromWire(buf, rdlen)
	default:
		return nil, fmt.Errorf("unimplement type: %v", t)
	}
}

func RdataFromString(t RRType, s string) (Rdata, error) {
	switch t {
	case RR_A:
		return AFromString(s)
	case RR_AAAA:
		return AAAAFromString(s)
	case RR_CNAME:
		return CNameFromString(s)
	case RR_SOA:
		return SOAFromString(s)
	case RR_NS:
		return NSFromString(s)
	case RR_OPT:
		return OPTFromString(s)
	case RR_PTR:
		return PTRFromString(s)
	case RR_SRV:
		return SRVFromString(s)
	case RR_NAPTR:
		return NAPTRFromString(s)
	case RR_DNAME:
		return DNameFromString(s)
	case RR_RRSIG:
		return RRSigFromString(s)
	case RR_MX:
		return MXFromString(s)
	case RR_TXT:
		return TxtFromString(s)
	case RR_RP:
		return RPFromString(s)
	case RR_SPF:
		return SPFFromString(s)
	case RR_NSEC3:
		return NSEC3FromString(s)
	case RR_DS:
		return DSFromString(s)
	default:
		return nil, fmt.Errorf("unimplement type: %v", t)
	}
}
