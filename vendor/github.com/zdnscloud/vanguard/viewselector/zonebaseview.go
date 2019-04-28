package viewselector

import (
	//"github.com/zdnscloud/g53"

	"github.com/zdnscloud/cement/domaintree"
	"github.com/zdnscloud/vanguard/core"
)

type ZoneBaseView struct {
	zoneBaseView *domaintree.DomainTree
	views        []string
}

func newZoneBaseView() *ZoneBaseView {
	zbv := &ZoneBaseView{}
	zbv.reload()
	return zbv
}

func (zbv *ZoneBaseView) reload() {
	zoneBaseView := domaintree.NewDomainTree()
	views := []string{}
	/*
		for _, zoneView := range zoneViews {
			zname, err := g53.NameFromString(zoneView.Zone)
			if err == nil {
				views = append(views, zoneView.View)
				zoneBaseView.Insert(zname, zoneView.View)
			} else {
				panic("invalid zone name:" + zoneView.Zone)
			}
		}
	*/
	zbv.zoneBaseView = zoneBaseView
	zbv.views = views
}

func (zbv *ZoneBaseView) ViewForQuery(client *core.Client) (string, bool) {
	qname := client.Request.Question.Name

	_, value, _ := zbv.zoneBaseView.Search(qname)
	if value != nil {
		return value.(string), true
	} else {
		return "", false
	}
}

func (zbv *ZoneBaseView) GetViews() []string {
	return zbv.views
}
