package g53

import (
	"github.com/zdnscloud/g53/util"
)

func MakeAXFR(zone *Name, tsig *TSIG) *Message {
	h := Header{}
	h.Opcode = OP_QUERY
	h.Id = util.GenMessageId()
	h.QDCount = 1
	q := &Question{
		Name:  zone,
		Type:  RR_AXFR,
		Class: CLASS_IN,
	}

	msg := &Message{
		Header:   h,
		Question: q,
	}
	msg.SetTSIG(tsig)
	return msg
}

func MakeIXFR(zone *Name, currentSOA *RRset, tsig *TSIG) *Message {
	h := Header{}
	h.Opcode = OP_QUERY
	h.Id = util.GenMessageId()
	h.QDCount = 1
	q := &Question{
		Name:  zone,
		Type:  RR_IXFR,
		Class: CLASS_IN,
	}

	h.NSCount = 1
	msg := &Message{
		Header:   h,
		Question: q,
	}
	msg.AddRRset(AuthSection, currentSOA)
	if tsig != nil {
		msg.SetTSIG(tsig)
	}
	return msg
}
