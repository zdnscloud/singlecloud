package auth

import (
	"github.com/zdnscloud/cement/domaintree"
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/logger"
	"github.com/zdnscloud/vanguard/resolver/auth/zone"
)

type Query struct {
	matchType   domaintree.SearchResult
	finder      zone.Zone
	request     *g53.Message
	response    *g53.Message
	answers     []*g53.RRset
	additionals []*g53.RRset
	authorities []*g53.RRset
}

func NewQuery(matchType domaintree.SearchResult, request *g53.Message, finder zone.Zone) *Query {
	return &Query{
		matchType: matchType,
		finder:    finder,
		request:   request,
		response:  request.MakeResponse(),
	}
}

func (q *Query) Process() {
	q.response.Header.SetFlag(g53.FLAG_AA, true)
	q.response.Header.Rcode = g53.R_NOERROR

	question := q.request.Question
	ctx := q.finder.Find(question.Name, question.Type, zone.DefaultFind)
	result := ctx.GetResult()
	switch result.Type {
	case zone.FRCname:
		logger.GetLogger().Debug("auth find cname")
		q.answers = append(q.answers, result.RRset)
	case zone.FRSuccess:
		logger.GetLogger().Debug("auth find exact rrset")
		q.answers = append(q.answers, result.RRset)
		q.additionals = append(q.additionals, ctx.GetAdditional()...)
		if q.matchType != domaintree.ExactMatch || question.Type != g53.RR_NS {
			q.addAuthAdditional()
		}
	case zone.FRDelegation:
		logger.GetLogger().Debug("auth find delegation")
		q.response.Header.SetFlag(g53.FLAG_AA, false)
		q.authorities = append(q.authorities, result.RRset)
		q.additionals = append(q.additionals, ctx.GetAdditional()...)
	case zone.FRNXDomain:
		logger.GetLogger().Debug("auth find no name")
		q.response.Header.Rcode = g53.R_NXDOMAIN
		q.addSOA()
	case zone.FRNXRRset:
		logger.GetLogger().Debug("auth find no rrset")
		q.addSOA()
	case zone.FRServFail:
		logger.GetLogger().Debug("auth find empty zone")
		q.response.Header.Rcode = g53.R_SERVFAIL
	default:
		panic("")
	}

	for _, rrset := range q.answers {
		q.response.AddRRset(g53.AnswerSection, rrset)
	}
	for _, rrset := range q.authorities {
		q.response.AddRRset(g53.AuthSection, rrset)
	}
	for _, rrset := range q.additionals {
		q.response.AddRRset(g53.AdditionalSection, rrset)
	}

	q.response.RecalculateSectionRRCount()
}

func (q *Query) addAuthAdditional() {
	ctx := q.finder.Find(q.finder.GetOrigin(), g53.RR_NS, zone.DefaultFind)
	result := ctx.GetResult()
	if result.Type != zone.FRSuccess {
		panic("zone short of apex ns")
	}
	q.authorities = append(q.authorities, result.RRset)
	q.additionals = append(q.additionals, ctx.GetAdditional()...)
}

func (q *Query) addSOA() {
	result := q.finder.Find(q.finder.GetOrigin(), g53.RR_SOA, zone.DefaultFind).GetResult()
	if result.Type != zone.FRSuccess {
		panic("zone short of soa")
	}
	q.authorities = append(q.authorities, result.RRset)
}

func (q *Query) GetResponse() *g53.Message {
	return q.response
}
