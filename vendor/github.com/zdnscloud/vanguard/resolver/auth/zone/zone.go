package zone

import (
	"errors"
	"net"

	"github.com/zdnscloud/g53"
)

var (
	ErrMultiRRForSingletonType    = errors.New("multiple RRs of singleton type")
	ErrWildcardNSRecord           = errors.New("ns record couldn't be wildcard")
	ErrOutOfZone                  = errors.New("rrset isn't contained in zone")
	ErrCNAMECoExistsWithOtherRR   = errors.New("cname and other type of RR don't coexist for same name")
	ErrShortOfSOA                 = errors.New("zone has no soa")
	ErrShortOfNS                  = errors.New("zone has no ns")
	ErrUnknownRRset               = errors.New("unknown rrset")
	ErrInvalidNSOwnerName         = errors.New("invalid NS owner name (wildcard)")
	ErrOriginNodeCouldNOTBeDelete = errors.New("zone name couldn't be removed")
	ErrUpdateSOA                  = errors.New("new SOA serial number is not greater than current SOA")
	ErrNoEffectiveUpdate          = errors.New("update has no impact")
	ErrSOANoAtTopZone             = errors.New("subdomain should not has SOA record")
	ErrServFail                   = errors.New("empty zone must servfail")
	ErrNoZonesUpdateAcls          = errors.New("no such zone for update acls")
	ErrNoZonesUpdateRole          = errors.New("no such zone for update role")
	ErrAbortLoad                  = errors.New("data invalid and abandon")
)

var SupportRRTypes = []g53.RRType{
	g53.RR_SOA,
	g53.RR_NS,
	g53.RR_A,
	g53.RR_AAAA,
	g53.RR_MX,
	g53.RR_SRV,
	g53.RR_SPF,
	g53.RR_PTR,
	g53.RR_TXT,
	g53.RR_CNAME,
	g53.RR_NAPTR,
	g53.RR_OPT,
	g53.RR_DNAME,
}

type ResultType int

const (
	FRSuccess    ResultType = 0
	FRDelegation ResultType = 1
	FRNXDomain   ResultType = 2
	FRNXRRset    ResultType = 3
	FRCname      ResultType = 4
	FRServFail   ResultType = 5
)

type FindResult struct {
	Type  ResultType
	RRset *g53.RRset
}

type FindOption int

const (
	DefaultFind FindOption = 0
	GlueOkFind  FindOption = 1
)

type FinderContext interface {
	GetResult() *FindResult
	GetAdditional() []*g53.RRset
}

type ZoneFinder interface {
	GetOrigin() *g53.Name
	Find(*g53.Name, g53.RRType, FindOption) FinderContext
}

type Transaction interface {
	RollBack() error
	Commit() error
}

type ZoneUpdator interface {
	Begin() (Transaction, error)
	Add(Transaction, *g53.RRset) error
	DeleteRRset(Transaction, *g53.RRset) error
	DeleteDomain(Transaction, *g53.Name) error
	DeleteRr(Transaction, *g53.RRset) error
	IncreaseSerialNumber(Transaction)
	Clean() error
}

type ZoneLoader interface {
	Load(<-chan *g53.RRset, <-chan struct{}) error
}

type ZoneTransfer interface {
	IsMaster() bool
	Masters() []string
	SetMasters([]string)
}

type SafeZone interface {
	GetUpdator(net.IP, bool) (ZoneUpdator, bool)
	SetAcls([]string)
}

type Zone interface {
	ZoneFinder
	ZoneLoader
	ZoneTransfer
	SafeZone
}

func IsRRsetTypeSupport(typ g53.RRType) bool {
	for _, typ_ := range SupportRRTypes {
		if typ == typ_ {
			return true
		}
	}
	return false
}
