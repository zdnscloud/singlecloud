package acl

import (
	"github.com/zdnscloud/vanguard/httpcmd"
)

var (
	ErrAnyNonAcl           = httpcmd.NewError(httpcmd.AclErrCodeStart, "any or none or all acl is read only")
	ErrAclExists           = httpcmd.NewError(httpcmd.AclErrCodeStart+1, "acl already exists")
	ErrNonExistAcl         = httpcmd.NewError(httpcmd.AclErrCodeStart+2, "operate non-exist acl")
	ErrAclInUse            = httpcmd.NewError(httpcmd.AclErrCodeStart+3, "delete acl is using by view")
	ErrAclInUseByView      = httpcmd.NewError(httpcmd.AclErrCodeStart+4, "acl is used by view")
	ErrAclInUseByAdZone    = httpcmd.NewError(httpcmd.AclErrCodeStart+5, "acl is used by ad zone")
	ErrAclInUseBySlaveZone = httpcmd.NewError(httpcmd.AclErrCodeStart+6, "acl is used by slave zone")
	ErrBadAclName          = httpcmd.NewError(httpcmd.AclErrCodeStart+7, "acl can't named with acl")
)
