package acl

import (
	"strings"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/vanguard/httpcmd"
)

type AddAcl struct {
	Name     string   `json:"name"`
	Networks []string `json:"networks"`
}

func (a *AddAcl) String() string {
	return "name: add acl and params: {name:" + a.Name + ", acls:[" +
		strings.Join(a.Networks, ",") + "]}"
}

type DeleteAcl struct {
	Name string `json:"name"`
}

func (a *DeleteAcl) String() string {
	return "name: delete acl and params: {name:" + a.Name + "}"
}

type UpdateAcl struct {
	Name     string   `json:"name"`
	Networks []string `json:"networks"`
}

func (a *UpdateAcl) String() string {
	return "name: update acl and params: {name:" + a.Name + ", acls:[" +
		strings.Join(a.Networks, ",") + "]}"
}

type GetAcls struct{}

func (g *GetAcls) String() string {
	return "name: get all acls"
}

type GetAcl struct {
	Name string `json:"name"`
}

func (a *GetAcl) String() string {
	return "name: get acl and params {name:" + a.Name + "}"
}

func (m *AclManager) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddAcl:
		return m.addAcl(c.Name, c.Networks)
	case *DeleteAcl:
		return m.deleteAcl(c.Name)
	case *UpdateAcl:
		return m.updateAcl(c.Name, c.Networks)
	default:
		panic("should not be here")
	}
}

func (m *AclManager) addAcl(name string, networks []string) (interface{}, *httpcmd.Error) {
	if err := checkNameValid(name); err != nil {
		return nil, err
	}

	if m.hasAcl(name) {
		return nil, ErrAclExists
	}

	return nil, m.add(name, networks)
}

func (m *AclManager) deleteAcl(name string) (interface{}, *httpcmd.Error) {
	if isReadOnly(name) {
		return nil, ErrAnyNonAcl
	}

	return nil, m.remove(name)
}

func (m *AclManager) updateAcl(name string, networks []string) (interface{}, *httpcmd.Error) {
	if isReadOnly(name) {
		return nil, ErrAnyNonAcl
	}

	return nil, m.update(name, networks)
}

func isReadOnly(name string) bool {
	return slice.SliceIndex([]string{AnyAcl, NoneAcl, AllAcl}, strings.ToLower(name)) != -1
}

func checkNameValid(name string) *httpcmd.Error {
	n := strings.ToLower(name)
	if n == BadName {
		return ErrBadAclName
	} else if isReadOnly(n) {
		return ErrAnyNonAcl
	} else {
		return nil
	}
}
