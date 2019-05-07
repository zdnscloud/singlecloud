package g53

import (
	"github.com/zdnscloud/g53/util"
)

func MakeUpdate(zone *Name) *Message {
	h := Header{}
	h.Opcode = OP_UPDATE
	h.Id = util.GenMessageId()

	q := &Question{
		Name:  zone,
		Type:  RR_SOA,
		Class: CLASS_IN,
	}

	return &Message{
		Header:   h,
		Question: q,
	}
}

//at least one rr with a specified name must exist
func (m *Message) UpdateNameExists(names []*Name) {
	for _, name := range names {
		m.AddRRset(AnswerSection, &RRset{
			Name:  name,
			Type:  RR_ANY,
			Class: CLASS_ANY,
			Ttl:   0,
		})
	}
}

// no rr of any type has specified name
func (m *Message) UpdateNameNotExists(names []*Name) {
	for _, name := range names {
		m.AddRRset(AnswerSection, &RRset{
			Name:  name,
			Type:  RR_ANY,
			Class: CLASS_NONE,
			Ttl:   0,
		})
	}
}

//rrset with specified rdata exists
func (m *Message) UpdateRdataExsits(rrset *RRset) {
	m.AddRRset(AnswerSection, rrset)
}

//rrset exists, (rr with name, type  exists)
func (m *Message) UpdateRRsetExists(rrset *RRset) {
	m.AddRRset(AnswerSection, &RRset{
		Name:  rrset.Name,
		Type:  rrset.Type,
		Class: CLASS_ANY,
		Ttl:   0,
	})
}

//rrset not exists,(rr with name type doesn't exists)
func (m *Message) UpdateRRsetNotExists(rrset *RRset) {
	m.AddRRset(AnswerSection, &RRset{
		Name:  rrset.Name,
		Type:  rrset.Type,
		Class: CLASS_NONE,
		Ttl:   0,
	})
}

// rrs are added
func (m *Message) UpdateAddRRset(rrset *RRset) {
	m.AddRRset(AuthSection, rrset)
}

// delete rrs with specified name, type
func (m *Message) UpdateRemoveRRset(rrset *RRset) {
	m.AddRRset(AuthSection, &RRset{
		Name:  rrset.Name,
		Type:  rrset.Type,
		Class: CLASS_ANY,
		Ttl:   0,
	})
}

// delete all rrset with name
func (m *Message) UpdateRemoveName(name *Name) {
	m.AddRRset(AuthSection, &RRset{
		Name:  name,
		Type:  RR_ANY,
		Class: CLASS_ANY,
		Ttl:   0,
	})
}

// Remove rr with specified rdata
func (m *Message) UpdateRemoveRdata(rrset *RRset) {
	m.AddRRset(AuthSection, &RRset{
		Name:   rrset.Name,
		Type:   rrset.Type,
		Class:  CLASS_NONE,
		Ttl:    0,
		Rdatas: rrset.Rdatas,
	})
}
