package viewselector

import (
	"github.com/zdnscloud/vanguard/httpcmd"
)

var (
	ErrTsigExists      = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart, "tsig key already exists")
	ErrViewExists      = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart+1, "view already exists")
	ErrModifyInnerView = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart+2, "update inner view")
	ErrConflictTsig    = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart+3, "conflict key secret")
	ErrNonExistTsig    = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart+4, "operate non-exist tsig key")
	ErrLessViewNumber  = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart+5, "modify view priority with less than existing number")
	ErrDefaultPriority = httpcmd.NewError(httpcmd.ViewSelectorErrCodeStart+6, "default priority should be lowest")
)
