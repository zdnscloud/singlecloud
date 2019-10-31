package auth

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/zdnscloud/cement/domaintree"
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
	"github.com/zdnscloud/vanguard/httpcmd"
	"github.com/zdnscloud/vanguard/logger"
	"github.com/zdnscloud/vanguard/resolver/auth/zone"
	"github.com/zdnscloud/vanguard/resolver/chain"
	"github.com/zdnscloud/vanguard/util"
	view "github.com/zdnscloud/vanguard/viewselector"
)

type AuthDataSource struct {
	chain.DefaultResolver
	viewZones map[string]*domaintree.DomainTree
	lock      sync.RWMutex
}

func NewAuth(conf *config.VanguardConf) *AuthDataSource {
	ds := &AuthDataSource{}
	ds.ReloadConfig(conf)
	httpcmd.RegisterHandler(ds, []httpcmd.Command{&AddAuthZone{}, &DeleteAuthZone{}, &UpdateAuthZone{}, &AddAuthRrs{}, &DeleteAuthRrs{}, &UpdateAuthRrs{}})
	return ds
}

func (ds *AuthDataSource) ReloadConfig(conf *config.VanguardConf) {
	viewZones := make(map[string]*domaintree.DomainTree)
	for view, _ := range view.GetViewAndIds() {
		viewZones[view] = domaintree.NewDomainTree()
	}

	for _, viewAuth := range conf.Auth {
		tree := viewZones[viewAuth.View]
		for _, z := range viewAuth.Zones {
			origin, err := g53.NameFromString(z.Name)
			if err != nil {
				panic("load auth zone " + z.Name + " failed:" + err.Error())
			}

			var zoneData zone.Zone
			if len(z.Masters) > 0 {
				zoneData = loadZoneFromMaster(origin, viewAuth.View, z.Masters)
				zoneData.SetMasters(z.Masters)
			} else {
				f, err := os.OpenFile(z.File, os.O_RDONLY, 0755)
				if err != nil {
					panic("open zone file " + z.File + " failed " + err.Error())
				}
				defer f.Close()
				content, err := ioutil.ReadAll(f)
				if err != nil {
					panic("read zone file " + z.File + " failed " + err.Error())
				}
				zoneData = loadZone(origin, string(content))
			}

			if _, err := tree.Insert(origin, zoneData); err != nil {
				panic("load auth zone " + z.Name + " failed:" + err.Error())
			} else {
				logger.GetLogger().Info("load zone %s in view %s succeed", z.Name, viewAuth.View)
			}
		}
	}

	ds.viewZones = viewZones
}

func (ds *AuthDataSource) Resolve(client *core.Client) {
	request := client.Request
	finder, matchType := ds.GetZone(client.View, request.Question.Name)
	if matchType == domaintree.NotFound {
		chain.PassToNext(ds, client)
		return
	}

	query := NewQuery(matchType, request, finder)
	query.Process()
	client.Response = query.GetResponse()
	if util.ClassifyResponse(client.Response) == util.REFERRAL {
		logger.GetLogger().Debug("auth found referral for %s in view %s",
			request.Question.String(), client.View)
		chain.PassToNext(ds, client)
	} else {
		client.CacheAnswer = false
	}
}

func (ds *AuthDataSource) GetZone(viewName string, name *g53.Name) (zone.Zone, domaintree.SearchResult) {
	ds.lock.RLock()
	zones, ok := ds.viewZones[viewName]
	ds.lock.RUnlock()
	if ok == false {
		return nil, domaintree.NotFound
	}

	_, z, result := zones.Search(name)
	if result != domaintree.NotFound {
		return z.(zone.Zone), result
	} else {
		return nil, result
	}
}
