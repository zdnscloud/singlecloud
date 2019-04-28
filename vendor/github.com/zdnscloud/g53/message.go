package g53

import (
	"bytes"

	"github.com/zdnscloud/g53/util"
)

type SectionType int

const (
	AnswerSection     SectionType = 0
	AuthSection       SectionType = 1
	AdditionalSection SectionType = 2
)

const SectionCount = 3

type Section []*RRset

func (section Section) rrCount() int {
	count := 0
	for _, rrset := range section {
		rrCount := rrset.RRCount()
		//for empty rdata, just count as 1
		if rrCount == 0 {
			rrCount = 1
		}
		count += rrCount
	}
	return count
}

func (s Section) Rend(r *MsgRender) {
	for _, rrset := range s {
		rrset.Rend(r)
	}
}

func (s Section) ToWire(buf *util.OutputBuffer) {
	for _, rrset := range s {
		rrset.ToWire(buf)
	}
}

func (s Section) String() string {
	var buf bytes.Buffer
	for _, rrset := range s {
		buf.WriteString(rrset.String())
	}
	return buf.String()
}

type Message struct {
	Header   Header
	Question *Question
	Sections [SectionCount]Section
	Edns     *EDNS
	Tsig     *TSIG
}

func MakeQuery(name *Name, typ RRType, size int, dnssec bool) *Message {
	h := Header{}
	h.SetFlag(FLAG_RD, true)
	h.Opcode = OP_QUERY
	h.Id = util.GenMessageId()
	h.QDCount = 1
	h.ARCount = 1

	q := &Question{
		Name:  name,
		Type:  typ,
		Class: CLASS_IN,
	}

	return &Message{
		Header:   h,
		Question: q,
		Edns: &EDNS{
			UdpSize:     uint16(size),
			DnssecAware: dnssec,
		},
	}
}

func MessageFromWire(buf *util.InputBuffer) (*Message, error) {
	m := Message{}
	if err := m.FromWire(buf); err != nil {
		return nil, err
	} else {
		return &m, nil
	}
}

func (m *Message) FromWire(buf *util.InputBuffer) error {
	h := &m.Header
	if err := HeaderFromWire(h, buf); err != nil {
		return err
	}

	if h.QDCount == 1 {
		q, err := QuestionFromWire(buf)
		if err != nil {
			return err
		}
		m.Question = q
	} else {
		m.Question = nil //in axfr message, question could be nil
	}

	m.Edns = nil
	m.Tsig = nil
	for i := 0; i < SectionCount; i++ {
		if err := m.sectionFromWire(SectionType(i), buf); err != nil {
			return err
		}
	}

	return nil
}

func (m *Message) sectionFromWire(st SectionType, buf *util.InputBuffer) error {
	var s Section
	var count uint16
	switch st {
	case AnswerSection:
		count = m.Header.ANCount
	case AuthSection:
		count = m.Header.NSCount
	case AdditionalSection:
		count = m.Header.ARCount
	}

	var lastRRset *RRset
	for i := uint16(0); i < count; i++ {
		rrset, err := RRsetFromWire(buf)
		if err != nil {
			return err
		}

		if lastRRset == nil {
			lastRRset = rrset
			continue
		}

		if lastRRset.IsSameRRset(rrset) {
			lastRRset.Rdatas = append(lastRRset.Rdatas, rrset.Rdatas[0])
		} else {
			s = append(s, lastRRset)
			lastRRset = rrset
		}
	}

	if lastRRset != nil {
		if st == AdditionalSection && lastRRset.Type == RR_OPT {
			m.Edns = EdnsFromRRset(lastRRset)
		} else if st == AdditionalSection && lastRRset.Type == RR_TSIG {
			m.Tsig = TSIGFromRRset(lastRRset)
		} else {
			s = append(s, lastRRset)
		}
	}

	m.Sections[st] = s
	return nil
}

func (m *Message) Rend(r *MsgRender) {
	(&m.Header).Rend(r)

	if m.Question != nil {
		m.Question.Rend(r)
	}

	for i := 0; i < SectionCount; i++ {
		m.Sections[i].Rend(r)
	}

	if m.Edns != nil {
		m.Edns.Rend(r)
	}

	if m.Tsig != nil {
		m.Tsig.Rend(r)
		r.WriteUint16At(uint16(m.Header.ARCount+1), 10)
	}
}

func (m *Message) RecalculateSectionRRCount() {
	if m.Question == nil {
		m.Header.QDCount = 0
	} else {
		m.Header.QDCount = 1
	}

	m.Header.ANCount = uint16(m.Sections[AnswerSection].rrCount())
	m.Header.NSCount = uint16(m.Sections[AuthSection].rrCount())
	m.Header.ARCount = uint16(m.Sections[AdditionalSection].rrCount())

	if m.Edns != nil {
		m.Header.ARCount += uint16(m.Edns.RRCount())
	}
}

func (m *Message) ToWire(buf *util.OutputBuffer) {
	(&m.Header).ToWire(buf)
	if m.Question != nil {
		m.Question.ToWire(buf)
	}

	for i := 0; i < SectionCount; i++ {
		m.Sections[i].ToWire(buf)
	}
}

func (m *Message) String() string {
	var buf bytes.Buffer
	buf.WriteString(m.Header.String())
	buf.WriteByte('\n')

	if m.Edns != nil {
		buf.WriteString(";; OPT PSEUDOSECTION:\n")
		buf.WriteString(m.Edns.String())
	}

	buf.WriteString(";; QUESTION SECTION:\n")
	if m.Question != nil {
		buf.WriteString(m.Question.String())
		buf.WriteByte('\n')
	}

	if len(m.Sections[AnswerSection]) > 0 {
		buf.WriteString("\n;; ANSWER SECTION:\n")
		buf.WriteString(m.Sections[AnswerSection].String())
	}

	if len(m.Sections[AuthSection]) > 0 {
		buf.WriteString("\n;; AUTHORITY SECTION:\n")
		buf.WriteString(m.Sections[AuthSection].String())
	}

	if len(m.Sections[AdditionalSection]) > 0 {
		buf.WriteString("\n;; ADDITIONAL SECTION:\n")
		buf.WriteString(m.Sections[AdditionalSection].String())
	}

	if m.Tsig != nil {
		buf.WriteString("\n;; TSIG PSEUDOSECTION:\n")
		buf.WriteString(m.Tsig.String())
	}

	return buf.String()
}

func (m *Message) GetSection(st SectionType) Section {
	return m.Sections[st]
}

func (m *Message) Clear() {
	m.Header.Clear()
	m.Question = nil
	for i := 0; i < SectionCount; i++ {
		m.Sections[i] = nil
	}
	m.Edns = nil
	m.Tsig = nil
}

func (m *Message) AddRRset(st SectionType, rrset *RRset) {
	m.Sections[st] = append(m.Sections[st], rrset)
}

func (m *Message) AddRR(st SectionType, name *Name, typ RRType, class RRClass, ttl RRTTL, rdata Rdata, merge bool) {
	if merge {
		if i := m.rrsetIndex(st, name, typ, class); i != -1 {
			m.Sections[st][i].AddRdata(rdata)
			m.Sections[st][i].Ttl = ttl
			return
		}
	}

	m.AddRRset(st, &RRset{
		Name:   name,
		Type:   typ,
		Class:  class,
		Ttl:    ttl,
		Rdatas: []Rdata{rdata},
	})
}

func (m *Message) HasRRset(st SectionType, rrset *RRset) bool {
	return m.rrsetIndex(st, rrset.Name, rrset.Type, rrset.Class) != -1
}

func (m *Message) rrsetIndex(st SectionType, name *Name, typ RRType, class RRClass) int {
	for i, rrset := range m.Sections[st] {
		if rrset.Class == class &&
			rrset.Type == typ &&
			rrset.Name.Equals(name) {
			return i
		}
	}
	return -1
}

func (m *Message) MakeResponse() *Message {
	h := Header{
		Id:      m.Header.Id,
		Opcode:  m.Header.Opcode,
		QDCount: m.Header.QDCount,
	}

	h.SetFlag(FLAG_QR, true)
	h.SetFlag(FLAG_RD, m.Header.GetFlag(FLAG_RD))

	return &Message{
		Header:   h,
		Question: m.Question,
	}
}

func (m *Message) ClearSection(s SectionType) {
	m.Sections[s] = nil
	switch s {
	case AnswerSection:
		m.Header.ANCount = 0
	case AuthSection:
		m.Header.NSCount = 0
	case AdditionalSection:
		m.Edns = nil
		m.Tsig = nil
		m.Header.ARCount = 0
	default:
		panic("question section couldn't be cleared")
	}
}

func (m *Message) SectionRRCount(s SectionType) int {
	return m.Sections[s].rrCount()
}

func (m *Message) SectionRRsetCount(s SectionType) int {
	return len(m.Sections[s])
}
