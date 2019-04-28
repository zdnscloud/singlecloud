package g53

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/g53/util"
)

type TSIGAlgorithm string

func AlgorithmFromString(name string) (TSIGAlgorithm, error) {
	switch strings.ToLower(name) {
	case "hmac-md5", "hmac-md5.sig-alg.reg.int.":
		return HmacMD5, nil
	case "hmac-sha1", "hmac-sha1.":
		return HmacSHA1, nil
	case "hmac-sha256", "hmac-sha256.":
		return HmacSHA256, nil
	case "hmac-sha512", "hmac-sha512.":
		return HmacSHA512, nil
	default:
		return "", errors.New("No such algorothm")
	}
}

const (
	HmacMD5    TSIGAlgorithm = "hmac-md5.sig-alg.reg.int."
	HmacSHA1   TSIGAlgorithm = "hmac-sha1."
	HmacSHA256 TSIGAlgorithm = "hmac-sha256."
	HmacSHA512 TSIGAlgorithm = "hmac-sha512."
)

var ErrSig = errors.New("signature error")
var ErrTime = errors.New("tsig time expired")

type TsigHeader struct {
	Name     *Name
	Rrtype   RRType
	Class    RRClass
	Ttl      RRTTL
	Rdlength uint16
}

func (h *TsigHeader) Rend(r *MsgRender) {
	h.Name.Rend(r)
	h.Rrtype.Rend(r)
	h.Class.Rend(r)
	h.Ttl.Rend(r)
	r.Skip(2)
}

func (h *TsigHeader) ToWire(buf *util.OutputBuffer) {
	h.Name.ToWire(buf)
	h.Rrtype.ToWire(buf)
	h.Class.ToWire(buf)
	h.Ttl.ToWire(buf)
	buf.Skip(2)
}

func (h *TsigHeader) String() string {
	var s []string
	s = append(s, h.Name.String(false))
	s = append(s, h.Ttl.String())
	s = append(s, h.Class.String())
	s = append(s, h.Rrtype.String())
	return strings.Join(s, "\t")
}

type TSIG struct {
	Header     *TsigHeader
	Algorithm  TSIGAlgorithm
	TimeSigned uint64
	Fudge      uint16
	MACSize    uint16
	MAC        []byte
	OrigId     uint16
	Error      uint16
	OtherLen   uint16
	OtherData  []byte
	hash       hash.Hash
}

func NewTSIG(key, secret string, alg string) (*TSIG, error) {
	name, err := NameFromString(key)
	if err != nil {
		return nil, err
	}

	algo, err := AlgorithmFromString(alg)
	if err != nil {
		return nil, err
	}

	h, err := hashSelect(algo, secret)
	if err != nil {
		return nil, err
	}

	return &TSIG{
		Header: &TsigHeader{
			Name:     name,
			Rrtype:   RR_TSIG,
			Class:    CLASS_ANY,
			Ttl:      0,
			Rdlength: 0,
		},
		Algorithm:  algo,
		TimeSigned: uint64(time.Now().Unix()),
		Fudge:      300,
		Error:      0,
		OtherLen:   0,
		hash:       h,
	}, nil
}

func (msg *Message) SetTSIG(tsig *TSIG) {
	if tsig != nil {
		tsig.OrigId = msg.Header.Id
	}
	msg.Tsig = tsig
}

func TSIGFromWire(buf *util.InputBuffer, ll uint16) (*TSIG, error) {
	i, ll, err := fieldFromWire(RDF_C_NAME, buf, ll)
	if err != nil {
		return nil, err
	}
	alg, _ := i.(*Name)
	algo := TSIGAlgorithm(alg.String(false))

	i, ll, err = fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	ts1, _ := i.(uint16)

	i, ll, err = fieldFromWire(RDF_C_UINT32, buf, ll)
	if err != nil {
		return nil, err
	}
	ts2, _ := i.(uint32)

	i, ll, err = fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	fudge, _ := i.(uint16)

	i, ll, err = fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	macSize, _ := i.(uint16)

	i, _, err = fieldFromWire(RDF_C_BINARY, buf, macSize)
	if err != nil {
		return nil, err
	}
	ll -= macSize
	mac, _ := i.([]byte)

	i, ll, err = fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	oid, _ := i.(uint16)

	i, ll, err = fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	erro, _ := i.(uint16)

	i, ll, err = fieldFromWire(RDF_C_UINT16, buf, ll)
	if err != nil {
		return nil, err
	}
	len, _ := i.(uint16)

	i, _, err = fieldFromWire(RDF_C_BINARY, buf, len)
	if err != nil {
		return nil, err
	}
	ll -= len
	odata, _ := i.([]byte)

	if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	}

	return &TSIG{
		Algorithm:  algo,
		TimeSigned: ((uint64(ts1) & 0x000000000000ffff) << 32) + uint64(ts2),
		Fudge:      fudge,
		MACSize:    macSize,
		MAC:        mac,
		OrigId:     oid,
		Error:      erro,
		OtherLen:   len,
		OtherData:  odata,
	}, nil
}

func TSIGFromRRset(rrset *RRset) *TSIG {
	tsig := rrset.Rdatas[0].(*TSIG)
	tsig.Header = &TsigHeader{
		Name:   rrset.Name,
		Rrtype: rrset.Type,
		Class:  rrset.Class,
		Ttl:    rrset.Ttl,
	}
	return tsig
}

func (t *TSIG) Rend(r *MsgRender) {
	t.genMessageHash(r.Data())
	t.Header.Rend(r)
	pos := r.Len()
	alg, _ := NameFromString(string(t.Algorithm))
	alg.Rend(r)
	ts1 := uint16((t.TimeSigned & 0x0000ffff00000000) >> 32)
	ts2 := uint32(t.TimeSigned & 0x00000000ffffffff)
	r.WriteUint16(ts1)
	r.WriteUint32(ts2)
	r.WriteUint16(t.Fudge)
	r.WriteUint16(t.MACSize)
	r.WriteData(t.MAC)
	r.WriteUint16(t.OrigId)
	r.WriteUint16(t.Error)
	r.WriteUint16(t.OtherLen)
	r.WriteData(t.OtherData)
	r.WriteUint16At(uint16(r.Len()-pos), pos-2)
}

func (t *TSIG) ToWire(buf *util.OutputBuffer) {
	t.Header.ToWire(buf)
	pos := buf.Len()
	alg, _ := NameFromString(string(t.Algorithm))
	alg.ToWire(buf)
	ts1 := uint16((t.TimeSigned & 0x0000ffff00000000) >> 32)
	ts2 := uint32(t.TimeSigned & 0x00000000ffffffff)
	buf.WriteUint16(ts1)
	buf.WriteUint32(ts2)
	buf.WriteUint16(t.Fudge)
	buf.WriteUint16(t.MACSize)
	buf.WriteData(t.MAC)
	buf.WriteUint16(t.OrigId)
	buf.WriteUint16(t.Error)
	buf.WriteUint16(t.OtherLen)
	buf.WriteData(t.OtherData)
	buf.WriteUint16At(uint16(buf.Len()-pos), pos-2)
}

func (t *TSIG) String() string {
	var s []string
	s = append(s, t.Header.String())
	s = append(s, "\t")
	s = append(s, string(t.Algorithm))
	s = append(s, tsigTimeToString(t.TimeSigned))
	s = append(s, strconv.Itoa(int(t.Fudge)))
	s = append(s, strconv.Itoa(int(t.MACSize)))
	s = append(s, strings.ToUpper(hex.EncodeToString(t.MAC)))
	s = append(s, strconv.Itoa(int(t.OrigId)))
	s = append(s, strconv.Itoa(int(t.Error)))
	s = append(s, strconv.Itoa(int(t.OtherLen)))
	s = append(s, string(t.OtherData))
	return strings.Join(s, " ")
}

func (t *TSIG) Compare(other Rdata) int {
	return 0
}

type tsigWireFmt struct {
	Name       *Name
	Class      RRClass
	Ttl        RRTTL
	Algorithm  TSIGAlgorithm
	TimeSigned uint64
	Fudge      uint16
	Error      uint16
	OtherLen   uint16
	OtherData  []byte
}

func (twf *tsigWireFmt) Rend(r *MsgRender) {
	twf.Name.Rend(r)
	twf.Class.Rend(r)
	twf.Ttl.Rend(r)
	alg, _ := NameFromString(string(twf.Algorithm))
	alg.Rend(r)
	ts1 := uint16((twf.TimeSigned & 0x0000ffff00000000) >> 32)
	ts2 := uint32(twf.TimeSigned & 0x00000000ffffffff)
	r.WriteUint16(ts1)
	r.WriteUint32(ts2)
	r.WriteUint16(twf.Fudge)
	r.WriteUint16(twf.Error)
	r.WriteUint16(twf.OtherLen)
	r.WriteData(twf.OtherData)
}

func (twf *tsigWireFmt) ToWire(buf *util.OutputBuffer) {
	twf.Name.ToWire(buf)
	twf.Class.ToWire(buf)
	twf.Ttl.ToWire(buf)
	alg, _ := NameFromString(string(twf.Algorithm))
	alg.ToWire(buf)
	ts1 := uint16((twf.TimeSigned & 0x0000ffff00000000) >> 32)
	ts2 := uint32(twf.TimeSigned & 0x00000000ffffffff)
	buf.WriteUint16(ts1)
	buf.WriteUint32(ts2)
	buf.WriteUint16(twf.Fudge)
	buf.WriteUint16(twf.Error)
	buf.WriteUint16(twf.OtherLen)
	buf.WriteData(twf.OtherData)
}

type macWirefmt struct {
	MACSize uint16
	MAC     []byte
}

func (mwf *macWirefmt) Rend(r *MsgRender) {
	r.WriteUint16(mwf.MACSize)
	r.WriteData(mwf.MAC)
}

func (mwf *macWirefmt) ToWire(buf *util.OutputBuffer) {
	buf.WriteUint16(mwf.MACSize)
	buf.WriteData(mwf.MAC)
}

func (tsig *TSIG) genMessageHash(messageRaw []byte) {
	if tsig.Error == 0 {
		buf := tsig.toWireFmtBuf(messageRaw, tsig.MAC)
		tsig.hash.Write(buf)
		tsig.MAC = tsig.hash.Sum(nil)
		tsig.MACSize = uint16(len(tsig.MAC))
	}
}

func (tsig *TSIG) VerifyTsig(msg *Message, secret string, requestMac []byte) error {
	msg.Tsig = nil
	render := NewMsgRender()
	msg.RecalculateSectionRRCount()
	msg.Rend(render)

	buf := tsig.toWireFmtBuf(render.Data(), requestMac)
	now := uint64(time.Now().Unix())
	ti := now - tsig.TimeSigned
	if now < tsig.TimeSigned {
		ti = tsig.TimeSigned - now
	}
	if uint64(tsig.Fudge) < ti {
		return ErrTime
	}

	h, err := hashSelect(tsig.Algorithm, secret)
	if err != nil {
		return err
	}

	h.Write(buf)
	if !hmac.Equal(h.Sum(nil), tsig.MAC) {
		return ErrSig
	}
	return nil
}

func (tsig *TSIG) toWireFmtBuf(msgBuf []byte, requestMac []byte) []byte {
	buf := util.NewOutputBuffer(512)

	if tsig.TimeSigned == 0 {
		tsig.TimeSigned = uint64(time.Now().Unix())
	}

	if tsig.Fudge == 0 {
		tsig.Fudge = 300
	}

	if requestMac != nil {
		(&macWirefmt{
			MACSize: uint16(len(requestMac)),
			MAC:     requestMac,
		}).ToWire(buf)
	}

	buf.WriteData(msgBuf)

	(&tsigWireFmt{
		Name:       tsig.Header.Name,
		Class:      CLASS_ANY,
		Ttl:        tsig.Header.Ttl,
		Algorithm:  tsig.Algorithm,
		TimeSigned: tsig.TimeSigned,
		Fudge:      tsig.Fudge,
		Error:      tsig.Error,
		OtherLen:   tsig.OtherLen,
		OtherData:  tsig.OtherData,
	}).ToWire(buf)

	return buf.Data()
}

func tsigTimeToString(t uint64) string {
	ti := time.Unix(int64(t), 0).UTC()
	return ti.Format("20060102150405")
}

func fromBase64(s []byte) ([]byte, error) {
	buflen := base64.StdEncoding.DecodedLen(len(s))
	buf := make([]byte, buflen)
	if n, err := base64.StdEncoding.Decode(buf, s); err != nil {
		return nil, err
	} else {
		return buf[:n], nil
	}
}

func hashSelect(algo TSIGAlgorithm, secret string) (hash.Hash, error) {
	rawsecret, err := fromBase64([]byte(secret))
	if err != nil {
		return nil, err
	}

	switch algo {
	case HmacMD5:
		return hmac.New(md5.New, rawsecret), nil
	case HmacSHA1:
		return hmac.New(sha1.New, rawsecret), nil
	case HmacSHA256:
		return hmac.New(sha256.New, rawsecret), nil
	case HmacSHA512:
		return hmac.New(sha512.New, rawsecret), nil
	default:
		panic("unknown algorithm")
	}
}
