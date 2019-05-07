package httpcmd

var (
	ErrUnknownView    = NewError(GeneralErrCodeStart, "view doesn't exists")
	ErrInvalidName    = NewError(GeneralErrCodeStart+1, "domain name isn't valid")
	ErrUnknownRRType  = NewError(GeneralErrCodeStart+2, "unknown rr type")
	ErrInvalidRR      = NewError(GeneralErrCodeStart+3, "rr isn't valid")
	ErrInvalidNetwork = NewError(GeneralErrCodeStart+4, "network isn't valid")
)
