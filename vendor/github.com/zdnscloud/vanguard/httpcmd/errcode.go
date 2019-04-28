package httpcmd

type Error struct {
	Code int    `json:"code"`
	Info string `json:"info"`
}

func (e *Error) Error() string {
	return e.Info
}

func NewError(code int, info string) *Error {
	return &Error{
		Code: code,
		Info: info,
	}
}

func (e *Error) AddDetail(info string) *Error {
	return NewError(e.Code, e.Info+":"+info)
}

const (
	InnerErrCodeStart         = 0
	GeneralErrCodeStart       = 100
	MetricErrCodeStart        = 200
	ServerErrCodeStart        = 300
	AclErrCodeStart           = 400
	ViewSelectorErrCodeStart  = 500
	AuthErrCodeStart          = 600
	CacheErrCodeStart         = 700
	ForwarderErrCodeStart     = 800
	RecursorErrCodeStart      = 900
	RateLimitErrCodeStart     = 1000
	StubZoneErrCodeStart      = 1100
	DNS64ErrCodeStart         = 1200
	LocalDataErrCodeStart     = 1300
	SortListErrCodeStart      = 1400
	FailForwarderErrCodeStart = 1500
	QuerySourceErrCodeStart   = 1600
)
