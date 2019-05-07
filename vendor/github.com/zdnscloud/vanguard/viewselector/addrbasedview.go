package viewselector

import (
	"sync"

	"github.com/zdnscloud/vanguard/acl"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
	"github.com/zdnscloud/vanguard/httpcmd"
)

type ViewAcls struct {
	name     string
	acls     []string
	priority int
}

type AddrBasedView struct {
	viewAcls []ViewAcls
	lock     sync.RWMutex
}

func newAddrBasedView() *AddrBasedView {
	v := &AddrBasedView{}
	httpcmd.RegisterHandler(v, []httpcmd.Command{&UpdateView{}, &UpdateViewPriority{}})
	return v
}

func (v *AddrBasedView) ReloadConfig(conf *config.VanguardConf) {
	var viewAcls []ViewAcls
	for i, viewAcl := range conf.Views.ViewAcls {
		viewAcls = append(viewAcls, ViewAcls{
			name:     viewAcl.View,
			acls:     viewAcl.Acls,
			priority: i,
		})
	}
	v.viewAcls = viewAcls
}

func (v *AddrBasedView) ViewForQuery(client *core.Client) (string, bool) {
	v.lock.RLock()
	defer v.lock.RUnlock()
	clientIP := client.IP()

	for _, viewAcl := range v.viewAcls {
		for _, aclName := range viewAcl.acls {
			if acl.GetAclManager().Find(aclName, clientIP) {
				return viewAcl.name, true
			}
		}
	}

	return "", false
}

func (v *AddrBasedView) GetViews() []string {
	var views []string
	for _, viewAcl := range v.viewAcls {
		views = append(views, viewAcl.name)
	}
	return views
}
