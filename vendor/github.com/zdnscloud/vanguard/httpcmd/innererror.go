package httpcmd

var (
	ErrBatchCmdNotSupport = NewError(InnerErrCodeStart, "batch cmd isn't supported")
	ErrUnknownCmd         = NewError(InnerErrCodeStart+1, "cmd is unknown")
	ErrCmdFormatInvalid   = NewError(InnerErrCodeStart+2, "command format isn't valid")
	ErrHTTPMethodInvalid  = NewError(InnerErrCodeStart+3, "http method shouldbe post")
	ErrAssertFailed       = NewError(InnerErrCodeStart+4, "assert failed, inner panic")
)
