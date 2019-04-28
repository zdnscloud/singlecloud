package viewselector

import (
	"errors"

	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
)

const (
	DefaultView = "default"
	AnyView     = "any"
)

var (
	errInvalidKey       = errors.New("No found available key in message with tsig")
	errNoMatchAlgorithm = errors.New("No match algorithm between configure and message with tsig")
	ErrNoAuthUpdate     = errors.New("No Auth to update zone")
)

var viewAndIds map[string]uint16

func GetViewAndIds() map[string]uint16 {
	return viewAndIds
}

type SelectorMgr struct {
	core.DefaultHandler
	selectors []ViewSelector
}

func NewSelectorMgr(conf *config.VanguardConf) core.DNSQueryHandler {
	mgr := &SelectorMgr{
		selectors: make([]ViewSelector, 0),
	}
	mgr.AddViewSelector(newAddrBasedView())
	mgr.AddViewSelector(newTSIGKeyBasedView())
	mgr.ReloadConfig(conf)
	return mgr
}

func (mgr *SelectorMgr) ReloadConfig(conf *config.VanguardConf) {
	for _, selector := range mgr.selectors {
		selector.ReloadConfig(conf)
	}
	mgr.allocateIdForView()
	return
}

func (mgr *SelectorMgr) AddViewSelector(s ViewSelector) {
	mgr.selectors = append(mgr.selectors, s)
}

func (mgr *SelectorMgr) HandleQuery(ctx *core.Context) {
	mgr.SelectView(ctx)
	core.PassToNext(mgr, ctx)
}

func (mgr *SelectorMgr) SelectView(ctx *core.Context) bool {
	view := ""
	for _, vs := range mgr.selectors {
		if v, found := vs.ViewForQuery(&ctx.Client); found {
			view = v
			break
		}
	}

	if view != "" {
		ctx.Client.View = view
		ctx.Client.ViewId = viewAndIds[view]
		return true
	} else {
		return false
	}
}

func (mgr *SelectorMgr) allocateIdForView() {
	viewAndIds = map[string]uint16{DefaultView: uint16(0)}
	id := uint16(1)

	for _, s := range mgr.selectors {
		for _, view := range s.GetViews() {
			if _, ok := viewAndIds[view]; ok == false {
				viewAndIds[view] = id
				id += 1
			}
		}
	}
}

//For testing
func InitViews(views ...string) {
	viewAndIds = make(map[string]uint16)
	for i, v := range views {
		viewAndIds[v] = uint16(i)
	}
}
