package g53

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zdnscloud/g53/util"
)

var (
	ErrUnknownRRType            = errors.New("unknown rr type")
	ErrUnknownRRClass           = errors.New("unknown rr class")
	ErrDuplicateRdata           = errors.New("duplicate rdata")
	ErrRRsetStringFormatInValid = errors.New("rrset string format isn't valid")
	ErrTtlFormatInvalid         = errors.New("ttl format isn't valid")
)

type RRTTL uint32
type RRClass uint16
type RRType uint16

const (
	CLASS_IN   RRClass = 1
	CLASS_CH   RRClass = 3
	CLASS_HS   RRClass = 4
	CLASS_NONE RRClass = 254
	CLASS_ANY  RRClass = 255
)

const (
	/** a host address */
	RR_A RRType = 1
	/** an authoritative name server */
	RR_NS RRType = 2
	/** the canonical name for an alias */
	RR_CNAME RRType = 5
	/**  marks the start of a zone of authority */
	RR_SOA RRType = 6
	/**  a mailbox domain name (EXPERIMENTAL) */
	RR_MB RRType = 7
	/**  a mail group member (EXPERIMENTAL) */
	RR_MG RRType = 8
	/**  a mail rename domain name (EXPERIMENTAL) */
	RR_MR RRType = 9
	/**  a null RR (EXPERIMENTAL) */
	RR_NULL RRType = 10
	/**  a well known service description */
	RR_WKS RRType = 11
	/**  a domain name pointer */
	RR_PTR RRType = 12
	/**  host information */
	RR_HINFO RRType = 13
	/**  mailbox or mail list information */
	RR_MINFO RRType = 14
	/**  mail exchange */
	RR_MX RRType = 15
	/**  text strings */
	RR_TXT RRType = 16
	/**  RFC1183 */
	RR_RP RRType = 17
	/**  RFC1183 */
	RR_AFSDB RRType = 18
	/**  RFC1183 */
	RR_X25 RRType = 19
	/**  RFC1183 */
	RR_ISDN RRType = 20
	/**  RFC1183 */
	RR_RT RRType = 21
	/**  RFC1706 */
	RR_NSAP RRType = 22
	/**  RFC1348 */
	RR_NSAP_PTR RRType = 23
	/**  2535typecode */
	RR_SIG RRType = 24
	/**  2535typecode */
	RR_KEY RRType = 25
	/**  RFC2163 */
	RR_PX RRType = 26
	/**  RFC1712 */
	RR_GPOS RRType = 27
	/**  ipv6 address */
	RR_AAAA RRType = 28
	/**  LOC record  RFC1876 */
	RR_LOC RRType = 29
	/**  2535typecode */
	RR_NXT RRType = 30
	/**  draft-ietf-nimrod-dns-01.txt */
	RR_EID RRType = 31
	/**  draft-ietf-nimrod-dns-01.txt */
	RR_NIMLOC RRType = 32
	/**  SRV record RFC2782 */
	RR_SRV RRType = 33
	/**  http://www.jhsoft.com/rfc/af-saa-0069.000.rtf */
	RR_ATMA RRType = 34
	/**  RFC2915 */
	RR_NAPTR RRType = 35
	/**  RFC2230 */
	RR_KX RRType = 36
	/**  RFC2538 */
	RR_CERT RRType = 37
	/**  RFC2874 */
	RR_A6 RRType = 38
	/**  RFC2672 */
	RR_DNAME RRType = 39
	/**  dnsind-kitchen-sink-02.txt */
	RR_SINK RRType = 40
	/**  Pseudo OPT record... */
	RR_OPT RRType = 41
	/**  RFC3123 */
	RR_APL RRType = 42
	/**  RFC4034 RFC3658 */
	RR_DS RRType = 43
	/**  SSH Key Fingerprint */
	RR_SSHFP RRType = 44 /* RFC 4255 */
	/**  IPsec Key */
	RR_IPSECKEY RRType = 45 /* RFC 4025 */
	/**  DNSSEC */
	RR_RRSIG  RRType = 46 /* RFC 4034 */
	RR_NSEC   RRType = 47 /* RFC 4034 */
	RR_DNSKEY RRType = 48 /* RFC 4034 */

	RR_DHCID RRType = 49 /* RFC 4701 */
	/* NSEC3 */
	RR_NSEC3      RRType = 50 /* RFC 5155 */
	RR_NSEC3PARAM RRType = 51 /* RFC 5155 */
	RR_TLSA       RRType = 52 /* RFC 6698 */

	RR_HIP RRType = 55 /* RFC 5205 */

	/** draft-reid-dnsext-zs */
	RR_NINFO RRType = 56
	/** draft-reid-dnsext-rkey */
	RR_RKEY RRType = 57
	/** draft-ietf-dnsop-trust-history */
	RR_TALINK RRType = 58
	/** draft-barwood-dnsop-ds-publis */
	RR_CDS RRType = 59

	RR_SPF RRType = 99 /* RFC 4408 */

	RR_UINFO  RRType = 100
	RR_UID    RRType = 101
	RR_GID    RRType = 102
	RR_UNSPEC RRType = 103

	RR_NID RRType = 104 /* RFC 6742 */
	RR_L32 RRType = 105 /* RFC 6742 */
	RR_L64 RRType = 106 /* RFC 6742 */
	RR_LP  RRType = 107 /* RFC 6742 */

	RR_EUI48 RRType = 108 /* RFC 7043 */
	RR_EUI64 RRType = 109 /* RFC 7043 */

	RR_TKEY RRType = 249 /* RFC 2930 */
	RR_TSIG RRType = 250
	RR_IXFR RRType = 251
	RR_AXFR RRType = 252
	/**  A request for mailbox-related records (MB MG or MR) */
	RR_MAILB RRType = 253
	/**  A request for mail agent RRs (Obsolete - see MX) */
	RR_MAILA RRType = 254
	/**  any type (wildcard) */
	RR_ANY RRType = 255
	/** draft-faltstrom-uri-06 */
	RR_URI RRType = 256
	RR_CAA RRType = 257 /* RFC 6844 */

	/** DNSSEC Trust Authorities */
	RR_TA RRType = 32768
	/* RFC 4431 5074 DNSSEC Lookaside Validation */
	RR_DLV RRType = 32769
)

var typeNameMap = map[RRType]string{
	RR_A:          "a",
	RR_NS:         "ns",
	RR_CNAME:      "cname",
	RR_SOA:        "soa",
	RR_MB:         "mb",
	RR_MG:         "mg",
	RR_MR:         "mr",
	RR_NULL:       "null",
	RR_WKS:        "wks",
	RR_PTR:        "ptr",
	RR_HINFO:      "hinfo",
	RR_MINFO:      "minfo",
	RR_MX:         "mx",
	RR_TXT:        "txt",
	RR_RP:         "rp",
	RR_AFSDB:      "afsdb",
	RR_X25:        "x25",
	RR_ISDN:       "isdn",
	RR_RT:         "rt",
	RR_NSAP:       "nsap",
	RR_NSAP_PTR:   "nsap-ptr",
	RR_SIG:        "sig",
	RR_KEY:        "key",
	RR_PX:         "px",
	RR_GPOS:       "gpos",
	RR_AAAA:       "aaaa",
	RR_LOC:        "loc",
	RR_NXT:        "nxt",
	RR_EID:        "eid",
	RR_NIMLOC:     "nimloc",
	RR_SRV:        "srv",
	RR_ATMA:       "atma",
	RR_NAPTR:      "naptr",
	RR_KX:         "kx",
	RR_CERT:       "cert",
	RR_A6:         "a6",
	RR_DNAME:      "dname",
	RR_SINK:       "sink",
	RR_OPT:        "opt",
	RR_APL:        "apl",
	RR_DS:         "ds",
	RR_SSHFP:      "sshfp",
	RR_IPSECKEY:   "ipseckey",
	RR_RRSIG:      "rrsig",
	RR_NSEC:       "nsec",
	RR_DNSKEY:     "dnskey",
	RR_DHCID:      "dhcid",
	RR_NSEC3:      "nsec3",
	RR_NSEC3PARAM: "nsec3param",
	RR_TLSA:       "tlsa",
	RR_HIP:        "hip",
	RR_NINFO:      "ninfo",
	RR_RKEY:       "pkey",
	RR_TALINK:     "talink",
	RR_CDS:        "cds",
	RR_SPF:        "spf",
	RR_UINFO:      "uinfo",
	RR_UID:        "uid",
	RR_GID:        "gid",
	RR_UNSPEC:     "unspec",
	RR_NID:        "nid",
	RR_L32:        "l32",
	RR_L64:        "l64",
	RR_LP:         "lp",
	RR_EUI48:      "eui48",
	RR_EUI64:      "eui64",

	RR_TKEY:  "tkey",
	RR_TSIG:  "tsig",
	RR_IXFR:  "ixfr",
	RR_AXFR:  "axfr",
	RR_MAILB: "mailb",
	RR_MAILA: "maila",
	RR_ANY:   "any",
	RR_URI:   "uri",
	RR_CAA:   "caa",
	RR_TA:    "ta",
	RR_DLV:   "dlv",
}

func TTLFromWire(buf *util.InputBuffer) (RRTTL, error) {
	ttl, err := buf.ReadUint32()
	if err != nil {
		return RRTTL(0), err
	}

	return RRTTL(ttl), nil
}

func TTLFromString(s string) (RRTTL, error) {
	ttl, err := strconv.Atoi(s)
	if err != nil {
		return RRTTL(0), ErrTtlFormatInvalid
	}

	return RRTTL(ttl), nil
}

func (ttl RRTTL) Rend(render *MsgRender) {
	render.WriteUint32(uint32(ttl))
}

func (ttl RRTTL) ToWire(buf *util.OutputBuffer) {
	buf.WriteUint32(uint32(ttl))
}

func (ttl RRTTL) String() string {
	return strconv.Itoa(int(ttl))
}

func ClassFromWire(buf *util.InputBuffer) (RRClass, error) {
	cls, err := buf.ReadUint16()
	if err != nil {
		return RRClass(0), err
	}

	return RRClass(cls), nil
}

func ClassFromString(s string) (RRClass, error) {
	s = strings.ToUpper(s)
	switch s {
	case "IN":
		return CLASS_IN, nil
	case "CH":
		return CLASS_CH, nil
	case "HS":
		return CLASS_HS, nil
	case "NONE":
		return CLASS_NONE, nil
	case "ANY":
		return CLASS_ANY, nil
	default:
		return RRClass(0), ErrUnknownRRClass
	}
}

func (cls RRClass) Rend(render *MsgRender) {
	render.WriteUint16(uint16(cls))
}

func (cls RRClass) ToWire(buf *util.OutputBuffer) {
	buf.WriteUint16(uint16(cls))
}

func (cls RRClass) String() string {
	switch cls {
	case CLASS_IN:
		return "IN"
	case CLASS_CH:
		return "CH"
	case CLASS_HS:
		return "HS"
	case CLASS_NONE:
		return "NONE"
	case CLASS_ANY:
		return "ANY"
	default:
		return "unknownclass"
	}
}

func TypeFromWire(buf *util.InputBuffer) (RRType, error) {
	t, err := buf.ReadUint16()
	if err != nil {
		return RRType(0), err
	}

	return RRType(t), nil
}

func TypeFromString(s string) (RRType, error) {
	s = strings.ToLower(s)
	for t, ts := range typeNameMap {
		if ts == s {
			return t, nil
		}
	}
	return RRType(0), ErrUnknownRRType
}

func (t RRType) Rend(render *MsgRender) {
	render.WriteUint16(uint16(t))
}

func (t RRType) ToWire(buf *util.OutputBuffer) {
	buf.WriteUint16(uint16(t))
}

func (t RRType) String() string {
	s := typeNameMap[t]
	if s == "" {
		return fmt.Sprintf("unknowntype:%d", t)
	} else {
		return strings.ToUpper(s)
	}
}

type RRset struct {
	Name   *Name
	Type   RRType
	Class  RRClass
	Ttl    RRTTL
	Rdatas []Rdata
}

//rrset string should be in format
//example.org. 300 IN NS ns.example.org.
var rrsetTemplate = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)\s*$`)

func RRsetFromString(s string) (*RRset, error) {
	fields := rrsetTemplate.FindStringSubmatch(s)
	if len(fields) != 6 {
		return nil, ErrRRsetStringFormatInValid
	}

	name, err := NameFromString(fields[1])
	if err != nil {
		return nil, err
	}

	ttl, err := TTLFromString(fields[2])
	if err != nil {
		return nil, err
	}

	cls, err := ClassFromString(fields[3])
	if err != nil {
		return nil, err
	}

	typ, err := TypeFromString(fields[4])
	if err != nil {
		return nil, err
	}

	rdata, err := RdataFromString(typ, fields[5])
	if err != nil {
		return nil, err
	}

	return &RRset{
		Name:   name,
		Type:   typ,
		Class:  cls,
		Ttl:    ttl,
		Rdatas: []Rdata{rdata},
	}, nil
}

func RRsetFromWire(buf *util.InputBuffer) (*RRset, error) {
	name, err := NameFromWire(buf, false)
	if err != nil {
		return nil, err
	}

	typ, err := TypeFromWire(buf)
	if err != nil {
		return nil, err
	}

	cls, err := ClassFromWire(buf)
	if err != nil {
		return nil, err
	}

	ttl, err := TTLFromWire(buf)
	if err != nil {
		return nil, err
	}

	rdata, err := RdataFromWire(typ, buf)
	if err != nil {
		return nil, err
	}

	var rdatas []Rdata
	if rdata != nil {
		rdatas = []Rdata{rdata}
	}

	return &RRset{
		Name:   name,
		Type:   typ,
		Class:  cls,
		Ttl:    ttl,
		Rdatas: rdatas,
	}, nil
}

func (rrset *RRset) Rend(r *MsgRender) {
	if len(rrset.Rdatas) == 0 {
		rrset.Name.Rend(r)
		rrset.Type.Rend(r)
		rrset.Class.Rend(r)
		rrset.Ttl.Rend(r)
		r.WriteUint16(0)
	} else {
		for _, rdata := range rrset.Rdatas {
			rrset.Name.Rend(r)
			rrset.Type.Rend(r)
			rrset.Class.Rend(r)
			rrset.Ttl.Rend(r)
			pos := r.Len()
			r.Skip(2)
			rdata.Rend(r)
			r.WriteUint16At(uint16(r.Len()-pos-2), pos)
		}
	}
}

func (rrset *RRset) ToWire(buf *util.OutputBuffer) {
	if len(rrset.Rdatas) == 0 {
		rrset.Name.ToWire(buf)
		rrset.Type.ToWire(buf)
		rrset.Class.ToWire(buf)
		rrset.Ttl.ToWire(buf)
		buf.WriteUint16(0)
	} else {
		for _, rdata := range rrset.Rdatas {
			rrset.Name.ToWire(buf)
			rrset.Type.ToWire(buf)
			rrset.Class.ToWire(buf)
			rrset.Ttl.ToWire(buf)

			pos := buf.Len()
			buf.Skip(2)
			rdata.ToWire(buf)
			buf.WriteUint16At(uint16(buf.Len()-pos-2), pos)
		}
	}
}

func (rrset *RRset) String() string {
	header := strings.Join([]string{rrset.Name.String(false), rrset.Ttl.String(), rrset.Class.String(), rrset.Type.String()}, "\t")
	if len(rrset.Rdatas) == 0 {
		return header
	} else {
		var buf bytes.Buffer
		for _, rdata := range rrset.Rdatas {
			buf.WriteString(header)
			buf.WriteString("\t")
			buf.WriteString(rdata.String())
			buf.WriteString("\n")
		}
		return buf.String()
	}
}

func (rrset *RRset) RRCount() int {
	return len(rrset.Rdatas)
}

func (rrset *RRset) IsSameRRset(other *RRset) bool {
	return (rrset.Type == other.Type) && rrset.Name.Equals(other.Name)
}

func (rrset *RRset) Equals(other *RRset) bool {
	if rrset.IsSameRRset(other) == false {
		return false
	}

	rdataCount := len(rrset.Rdatas)
	if rdataCount != len(other.Rdatas) {
		return false
	}

	if rdataCount == 0 {
		return true
	}

	selfClone := rrset.Clone()
	otherClone := other.Clone()
	selfClone.SortRdata()
	otherClone.SortRdata()
	for i := 0; i < rdataCount; i++ {
		if selfClone.Rdatas[i].Compare(otherClone.Rdatas[i]) != 0 {
			return false
		}
	}
	return true
}

func (rrset *RRset) AddRdata(rdata Rdata) error {
	for _, oldRdata := range rrset.Rdatas {
		if oldRdata.Compare(rdata) == 0 {
			return ErrDuplicateRdata
		}
	}

	rrset.Rdatas = append(rrset.Rdatas, rdata)
	return nil
}

func (rrset *RRset) RemoveRdata(rdata Rdata) bool {
	for i, oldRdata := range rrset.Rdatas {
		if oldRdata.Compare(rdata) == 0 {
			rdatas := rrset.Rdatas
			if len(rdatas) == 1 {
				rrset.Rdatas = nil
			} else {
				rrset.Rdatas = append(rdatas[:i], rdatas[i+1:]...)
			}
			return true
		}
	}

	return false
}

func (rrset *RRset) RotateRdata() {
	rrCount := rrset.RRCount()
	if rrCount < 2 {
		return
	}

	rrset.Rdatas = append([]Rdata{rrset.Rdatas[rrCount-1]}, rrset.Rdatas[0:rrCount-1]...)
}

type RdataSlice []Rdata

func (rdatas RdataSlice) Len() int           { return len(rdatas) }
func (rdatas RdataSlice) Swap(i, j int)      { rdatas[i], rdatas[j] = rdatas[j], rdatas[i] }
func (rdatas RdataSlice) Less(i, j int) bool { return rdatas[i].Compare(rdatas[j]) < 0 }

func (rrset *RRset) SortRdata() {
	sort.Sort(RdataSlice(rrset.Rdatas))
}

func (rrset *RRset) Clone() *RRset {
	rdataCount := len(rrset.Rdatas)
	rdatas := make([]Rdata, rdataCount, rdataCount)
	copy(rdatas, rrset.Rdatas)
	return &RRset{
		Name:   rrset.Name,
		Type:   rrset.Type,
		Class:  rrset.Class,
		Ttl:    rrset.Ttl,
		Rdatas: rdatas,
	}
}
