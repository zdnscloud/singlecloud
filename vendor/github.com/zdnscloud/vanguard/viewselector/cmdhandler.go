package viewselector

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/zdnscloud/vanguard/httpcmd"
)

type UpdateView struct {
	Name string   `json:"name"`
	Acls []string `json:"acls"`
}

func (v *UpdateView) String() string {
	return "name: update view and params {name:" + v.Name +
		", acls:[" + strings.Join(v.Acls, ",") + "]}"
}

type UpdateViewPriority struct {
	Orders []string `json:"orders"`
}

func (v *UpdateViewPriority) String() string {
	return "name: update view priority and params: {orders:[" +
		strings.Join(v.Orders, ",") + "]}"
}

func (v *AddrBasedView) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *UpdateView:
		return v.updateView(c.Name, c.Acls)
	case *UpdateViewPriority:
		return v.updateViewPriority(c.Orders)
	default:
		panic("should not be here")
	}
}

func (v *AddrBasedView) updateView(name string, acls []string) (interface{}, *httpcmd.Error) {
	name = strings.ToLower(name)
	if name == AnyView {
		return nil, ErrModifyInnerView
	}

	v.lock.Lock()
	for i := 0; i < len(v.viewAcls); i++ {
		if v.viewAcls[i].name == name {
			v.viewAcls[i].acls = acls
			break
		}
	}
	v.lock.Unlock()
	return nil, nil
}

type ViewByPriority []ViewAcls

func (v ViewByPriority) Len() int           { return len(v) }
func (v ViewByPriority) Less(i, j int) bool { return v[i].priority < v[j].priority }
func (v ViewByPriority) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

func (v *AddrBasedView) updateViewPriority(orders []string) (interface{}, *httpcmd.Error) {
	viewPriorities := make(map[string]int)
	for i, view := range orders {
		viewPriorities[view] = i
	}

	newViews := v.viewAcls
	var hasView bool
	for i := 0; i < len(newViews); i++ {
		newViews[i].priority, hasView = viewPriorities[newViews[i].name]
		if hasView == false {
			return nil, httpcmd.ErrUnknownView.AddDetail(newViews[i].name)
		}
	}
	sort.Sort(ViewByPriority(newViews))

	v.lock.Lock()
	v.viewAcls = newViews
	v.lock.Unlock()
	return nil, nil
}

func parseBindIps(ipStrs []string) ([]net.IP, error) {
	var bindIPs []net.IP
	for _, ipStr := range ipStrs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid bind ip address %s", ipStr)
		}
		bindIPs = append(bindIPs, ip)
	}

	return bindIPs, nil
}
