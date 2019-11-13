package auth

import (
	"bytes"
	"strings"

	"github.com/zdnscloud/cement/domaintree"
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/httpcmd"
	zn "github.com/zdnscloud/vanguard/resolver/auth/zone"
)

type AddAuthZone struct {
	View    string   `json:"view"`
	Name    string   `json:"name"`
	Masters []string `json:"masters"`
	Content string   `json:"zone_content"`
}

func (z *AddAuthZone) String() string {
	return "name: add authzone and params: {zone:" + z.Name +
		", view:" + z.View +
		", masters:" + strings.Join(z.Masters, ",") +
		", zone_content:" + z.Content + "]}"
}

type DeleteAuthZone struct {
	View string `json:"view"`
	Name string `json:"name"`
}

func (z *DeleteAuthZone) String() string {
	return "name: delete authzone and params: {zone:" + z.Name +
		", view:" + z.View + "}"
}

type UpdateAuthZone struct {
	View    string   `json:"view"`
	Name    string   `json:"name"`
	Masters []string `json:"masters"`
}

func (z *UpdateAuthZone) String() string {
	return "name: update authzone and params: {zone:" + z.Name +
		", view:" + z.View +
		", masters:" + strings.Join(z.Masters, ",") + "]}"
}

type AuthRR struct {
	View  string
	Zone  string
	Name  string
	Ttl   string
	Type  string
	Rdata string
}

type AuthRRs []*AuthRR

type AddAuthRrs struct {
	Rrs AuthRRs `json:"rrs"`
}

func (z *AddAuthRrs) String() string {
	return "name: add auth rrs and params: {\n" + stringFromRRs(z.Rrs) + "}"
}

func stringFromRRs(rrs AuthRRs) string {
	var buf bytes.Buffer
	for _, rr := range rrs {
		buf.WriteString("view:")
		buf.WriteString(rr.View)
		buf.WriteString(", zone:")
		buf.WriteString(rr.Zone)
		buf.WriteString(", name:")
		buf.WriteString(rr.Name)
		buf.WriteString(", ttl:")
		buf.WriteString(rr.Ttl)
		buf.WriteString(", type:")
		buf.WriteString(rr.Type)
		buf.WriteString(", rdata:")
		buf.WriteString(rr.Rdata)
		buf.WriteString("\n")
	}
	return buf.String()
}

type DeleteAuthRrs struct {
	Rrs AuthRRs `json:"rrs"`
}

func (z *DeleteAuthRrs) String() string {
	return "name: delete auth rrs and params: {\n" + stringFromRRs(z.Rrs) + "}"
}

type UpdateAuthRrs struct {
	OldRrs AuthRRs `json:"old_rrs"`
	NewRrs AuthRRs `json:"new_rrs"`
}

func (z *UpdateAuthRrs) String() string {
	return "name: update auth rr ttl or rdata and params: {rrs for delete:\n" + stringFromRRs(z.OldRrs) +
		", rrs for add:\n" + stringFromRRs(z.NewRrs) + "}"
}

func (z *AuthDataSource) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddAuthZone:
		return nil, z.addAuthZone(c.View, c.Name, c.Content, c.Masters)
	case *DeleteAuthZone:
		return nil, z.deleteAuthZone(c.View, c.Name)
	case *UpdateAuthZone:
		return nil, z.updateAuthZone(c.View, c.Name, c.Masters)
	case *AddAuthRrs:
		return nil, z.addAuthRrs(c.Rrs)
	case *DeleteAuthRrs:
		return nil, z.deleteAuthRrs(c.Rrs)
	case *UpdateAuthRrs:
		return nil, z.updateAuthRrs(c.OldRrs, c.NewRrs)
	default:
		panic("should not be here")
	}
}

func (z *AuthDataSource) addAuthZone(view, name, content string, masters []string) *httpcmd.Error {
	if err := z.loadZoneData(g53.NameFromStringUnsafe(name), view, content, masters); err != nil {
		return ErrInvalidZoneData.AddDetail(err.Error())
	} else {
		return nil
	}
}

func (z *AuthDataSource) deleteAuthZone(view, name string) *httpcmd.Error {
	origin := g53.NameFromStringUnsafe(name)
	_, result := z.GetZone(view, origin)
	if result != domaintree.ExactMatch {
		return ErrGetZoneFail
	}

	z.lock.Lock()
	z.viewZones[view].Delete(origin)
	z.lock.Unlock()
	return nil
}

func (z *AuthDataSource) updateAuthZone(view, name string, masters []string) *httpcmd.Error {
	origin := g53.NameFromStringUnsafe(name)
	zoneData, result := z.GetZone(view, origin)
	if result != domaintree.ExactMatch {
		return ErrGetZoneFail
	}

	zoneData.SetMasters(masters)
	if len(masters) != 0 {
		zoneData = loadZoneFromMaster(origin, view, masters)
	}
	z.lock.Lock()
	z.viewZones[view].Delete(origin)
	_, err := z.viewZones[view].Insert(origin, zoneData)
	z.lock.Unlock()

	if err != nil {
		return ErrUpdateZoneFailed.AddDetail(err.Error())
	} else {
		return nil
	}
}

func (z *AuthDataSource) addAuthRrs(rrs AuthRRs) *httpcmd.Error {
	newRRsets := make([]*g53.RRset, 0, len(rrs))
	var targetView string
	var targetZone *g53.Name
	for _, rr := range rrs {
		origin, err := g53.NameFromString(rr.Zone)
		if err != nil {
			return ErrInvalidZoneName.AddDetail(err.Error())
		}

		rrset, err := newRRset(rr.Name, rr.Ttl, rr.Type, rr.Rdata, g53.CLASS_IN)
		if err != nil {
			return ErrInvalidRR.AddDetail(err.Error())
		}

		if targetView == "" {
			targetView = rr.View
			targetZone = origin
		} else if targetView != rr.View {
			return ErrZoneUpdateFailed.AddDetail("add rr in different view in one request")
		} else if targetZone.Equals(origin) == false {
			return ErrZoneUpdateFailed.AddDetail("add rr in different zone in one request")
		}
		newRRsets = append(newRRsets, rrset)
	}

	if err := z.handleDynamicRRsets(targetView, targetZone, nil, newRRsets); err != nil {
		return ErrZoneUpdateFailed.AddDetail(err.Error())
	} else {
		return nil
	}
}

func (z *AuthDataSource) deleteAuthRrs(rrs AuthRRs) *httpcmd.Error {
	rrsetsToRemove := make([]*g53.RRset, 0, len(rrs))
	var targetView string
	var targetZone *g53.Name
	for _, rr := range rrs {
		origin, err := g53.NameFromString(rr.Zone)
		if err != nil {
			return ErrInvalidZoneName.AddDetail(err.Error())
		}

		rrset, err := newRRset(rr.Name, rr.Ttl, rr.Type, rr.Rdata, g53.CLASS_NONE)
		if err != nil {
			return ErrInvalidRR.AddDetail(err.Error())
		}

		if targetView == "" {
			targetView = rr.View
			targetZone = origin
		} else if targetView != rr.View {
			return ErrZoneUpdateFailed.AddDetail("delete rr in different view in one request")
		} else if targetZone.Equals(origin) == false {
			return ErrZoneUpdateFailed.AddDetail("delete rr in different zone in one request")
		}

		rrsetsToRemove = append(rrsetsToRemove, rrset)
	}

	if err := z.handleDynamicRRsets(targetView, targetZone, nil, rrsetsToRemove); err != nil {
		return ErrZoneUpdateFailed.AddDetail(err.Error())
	}

	return nil
}

func (z *AuthDataSource) updateAuthRrs(oldRrs, newRrs AuthRRs) *httpcmd.Error {
	if len(oldRrs) != 0 && oldRrs[0].Type != g53.RR_SOA.String() && oldRrs[0].Type != g53.RR_CNAME.String() {
		if err := z.deleteAuthRrs(oldRrs); err != nil {
			return err
		}
	}

	return z.addAuthRrs(newRrs)
}

func genBasicZoneContent(origin string) string {
	var buf bytes.Buffer
	buf.WriteString(origin)
	buf.WriteString(" 3600 IN SOA ns.")
	buf.WriteString(origin)
	buf.WriteString(" mail.")
	buf.WriteString(origin)
	buf.WriteString(" 1 28800 3600 604800 1800\n")
	buf.WriteString(origin)
	buf.WriteString(" 3600 IN NS ns.")
	buf.WriteString(origin)
	buf.WriteString("\n")
	buf.WriteString("ns.")
	buf.WriteString(origin)
	buf.WriteString(" 3600 IN A 127.0.0.1\n")
	return buf.String()
}

func (z *AuthDataSource) loadZoneData(origin *g53.Name, viewName, content string, masters []string) error {
	tree, ok := z.viewZones[viewName]
	if ok == false {
		return httpcmd.ErrUnknownView.AddDetail(viewName)
	}

	var zoneData zn.Zone
	if len(masters) != 0 {
		zoneData = loadZoneFromMaster(origin, viewName, masters)
	} else {
		if content == "" {
			content = genBasicZoneContent(origin.String(false))
		}
		zoneData = loadZone(origin, content)
	}

	z.lock.Lock()
	_, err := tree.Insert(origin, zoneData)
	z.lock.Unlock()

	return err
}

func newRRset(name, ttl, typ, rdata string, class g53.RRClass) (*g53.RRset, error) {
	rrName, err := g53.NameFromString(name)
	if err != nil {
		return nil, err
	}

	var rrTtl g53.RRTTL
	if ttl == "" {
		rrTtl = 0
	} else {
		rrTtl, err = g53.TTLFromString(ttl)
		if err != nil {
			return nil, err
		}
	}

	rrType, err := g53.TypeFromString(typ)
	if err != nil {
		return nil, err
	}

	rrData, err := g53.RdataFromString(rrType, rdata)
	if err != nil {
		return nil, err
	}

	return &g53.RRset{
		Name:   rrName,
		Type:   rrType,
		Ttl:    rrTtl,
		Class:  class,
		Rdatas: []g53.Rdata{rrData},
	}, nil
}
