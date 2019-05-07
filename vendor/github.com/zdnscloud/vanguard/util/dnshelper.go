package util

import (
	"strings"

	"github.com/zdnscloud/g53"
)

type AnswerType int

const (
	EMPTY    AnswerType = 0
	ANSWER   AnswerType = 1 //which include cname
	REFERRAL AnswerType = 2
	NXDOMAIN AnswerType = 3
	NXRRSET  AnswerType = 4
	UNKNOWN  AnswerType = 10
)

func ClassifyResponse(response *g53.Message) AnswerType {
	switch response.Header.Rcode {
	case g53.R_NXDOMAIN:
		return NXDOMAIN
	case g53.R_NOERROR:
		if response.Header.ANCount > 0 {
			firstRRsetInAnswer := response.Sections[g53.AnswerSection][0]
			if firstRRsetInAnswer.Name.Equals(response.Question.Name) {
				if firstRRsetInAnswer.Type == response.Question.Type ||
					firstRRsetInAnswer.Type == g53.RR_CNAME {
					return ANSWER
				}
			}
		} else if response.Header.NSCount > 0 {
			authRRset := response.Sections[g53.AuthSection][0]
			switch authRRset.Type {
			case g53.RR_NS:
				if response.Question.Name.IsSubDomain(authRRset.Name) {
					return REFERRAL
				}
			case g53.RR_SOA:
				return NXRRSET
			}
		}

	}
	return UNKNOWN
}

func NameStripFirstWildcard(name string) (bool, *g53.Name, error) {
	hasWildcard := false
	if strings.HasPrefix(name, "*.") {
		name = name[2:]
		hasWildcard = true
	}
	dname, err := g53.NameFromString(name)
	return hasWildcard, dname, err
}
