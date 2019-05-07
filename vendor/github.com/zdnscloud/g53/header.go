package g53

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/zdnscloud/g53/util"
)

type HeaderFlag uint16
type FlagField uint16

const (
	FLAG_QR FlagField = 0x8000
	FLAG_AA FlagField = 0x0400
	FLAG_TC FlagField = 0x0200
	FLAG_RD FlagField = 0x0100
	FLAG_RA FlagField = 0x0080
	FLAG_AD FlagField = 0x0020
	FLAG_CD FlagField = 0x0010
)

const (
	HEADERFLAG_MASK uint16 = 0x87b0
	OPCODE_MASK     uint16 = 0x7800
	OPCODE_SHIFT    uint16 = 11
	RCODE_MASK      uint16 = 0x000f
)

type Header struct {
	Id      uint16
	Flag    HeaderFlag
	Opcode  Opcode
	Rcode   Rcode
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

func (h *Header) Clear() {
	h.Flag = 0
	h.QDCount = 0
	h.ANCount = 0
	h.NSCount = 0
	h.ARCount = 0
}

func (h *Header) GetFlag(ff FlagField) bool {
	return (uint16(h.Flag) & uint16(ff)) != 0
}

func (h *Header) SetFlag(ff FlagField, set bool) {
	if set {
		h.Flag = HeaderFlag(uint16(h.Flag) | uint16(ff))
	} else {
		h.Flag = HeaderFlag(uint16(h.Flag) & uint16(^ff))
	}
}

func HeaderFromWire(h *Header, buf *util.InputBuffer) error {
	if buf.Len() < 12 {
		return errors.New("too short wire data for message header")
	}
	h.Id, _ = buf.ReadUint16()
	flag, _ := buf.ReadUint16()
	h.Flag = HeaderFlag(flag & HEADERFLAG_MASK)
	h.Opcode = Opcode((flag & OPCODE_MASK) >> OPCODE_SHIFT)
	h.Rcode = Rcode(flag & RCODE_MASK)
	h.QDCount, _ = buf.ReadUint16()
	h.ANCount, _ = buf.ReadUint16()
	h.NSCount, _ = buf.ReadUint16()
	h.ARCount, _ = buf.ReadUint16()
	return nil
}

func (h *Header) Rend(r *MsgRender) {
	r.WriteUint16(h.Id)
	r.WriteUint16(h.flag())
	r.WriteUint16(h.QDCount)
	r.WriteUint16(h.ANCount)
	r.WriteUint16(h.NSCount)
	r.WriteUint16(h.ARCount)
}

func (h *Header) flag() uint16 {
	flag := (uint16(h.Opcode) << OPCODE_SHIFT) & OPCODE_MASK
	flag |= uint16(h.Rcode) & RCODE_MASK
	flag |= uint16(h.Flag) & HEADERFLAG_MASK
	return flag
}

func (h *Header) ToWire(buf *util.OutputBuffer) {
	buf.WriteUint16(h.Id)
	buf.WriteUint16(h.flag())
	buf.WriteUint16(h.QDCount)
	buf.WriteUint16(h.ANCount)
	buf.WriteUint16(h.NSCount)
	buf.WriteUint16(h.ARCount)
}

func (h *Header) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(";; ->>HEADER<<- opcode: %s, status: %s, id: %d\n", h.Opcode.String(), h.Rcode.String(), h.Id))
	buf.WriteString(";; flags: ")
	if h.GetFlag(FLAG_QR) {
		buf.WriteString(" qr")
	}

	if h.GetFlag(FLAG_AA) {
		buf.WriteString(" aa")
	}

	if h.GetFlag(FLAG_TC) {
		buf.WriteString(" tc")
	}

	if h.GetFlag(FLAG_RD) {
		buf.WriteString(" rd")
	}

	if h.GetFlag(FLAG_RA) {
		buf.WriteString(" ra")
	}

	if h.GetFlag(FLAG_AD) {
		buf.WriteString(" ad")
	}

	if h.GetFlag(FLAG_CD) {
		buf.WriteString(" cd")
	}
	buf.WriteString("; ")

	buf.WriteString(fmt.Sprintf("QUERY: %d, ", h.QDCount))
	buf.WriteString(fmt.Sprintf("ANSWER: %d, ", h.ANCount))
	buf.WriteString(fmt.Sprintf("AUTHORITY: %d, ", h.NSCount))
	buf.WriteString(fmt.Sprintf("ADDITIONAL: %d, ", h.ARCount))
	buf.WriteString("\n")
	return buf.String()
}
