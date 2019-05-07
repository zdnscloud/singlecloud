package auth

import (
	"net"

	"github.com/zdnscloud/cement/domaintree"
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/core"
	"github.com/zdnscloud/vanguard/logger"
	"github.com/zdnscloud/vanguard/resolver/auth/zone"
	view "github.com/zdnscloud/vanguard/viewselector"
)

func (ds *AuthDataSource) HandleUpdate(ctx *core.Context) {
	if err := ds.handleUpdate(ctx); err != nil {
		if err == zone.ErrServFail {
			ctx.Client.Response.Header.Rcode = g53.R_SERVFAIL
		} else {
			ctx.Client.Response.Header.Rcode = g53.R_NOTAUTH
		}
		logger.GetLogger().Error("Update failed: %s", err.Error())
	}
}

func (ds *AuthDataSource) handleUpdate(ctx *core.Context) error {
	client := &ctx.Client
	return ds.handleDynamicRRsets(client.View,
		client.Request.Question.Name,
		client.IP(),
		client.Request.GetSection(g53.AuthSection))
}

func (ds *AuthDataSource) handleDynamicRRsets(viewName string, zoneName *g53.Name, clientIP net.IP, rrsets []*g53.RRset) error {
	updator, err := ds.getUpdator(viewName, zoneName, clientIP)
	if err != nil {
		return err
	}

	tx, err := updator.Begin()
	if err != nil {
		return view.ErrNoAuthUpdate
	}

	explicitUpdateSOA := false
	hasRRModified := false
	for _, rrset := range rrsets {
		if rrset.Class == g53.CLASS_IN {
			err = updator.Add(tx, rrset)
		} else {
			if rrset.Type == g53.RR_ANY {
				err = updator.DeleteDomain(tx, rrset.Name)
			} else if rrset.Class == g53.CLASS_ANY {
				err = updator.DeleteRRset(tx, rrset)
			} else {
				err = updator.DeleteRr(tx, rrset)
			}
		}

		if err == nil {
			if rrset.Type == g53.RR_SOA {
				explicitUpdateSOA = true
			}
			hasRRModified = true
		} else if err != zone.ErrNoEffectiveUpdate {
			logger.GetLogger().Error("update rr failed: %s, %s", err.Error(), rrset.String())
			tx.RollBack()
			return err
		}
	}

	if hasRRModified == false {
		logger.GetLogger().Warn("update does nothing")
		tx.RollBack()
		return nil
	} else {
		if explicitUpdateSOA == false {
			updator.IncreaseSerialNumber(tx)
		}
		return tx.Commit()
	}
}

func (ds *AuthDataSource) getUpdator(viewName string, origin *g53.Name, clientIP net.IP) (zone.ZoneUpdator, error) {
	zone, result := ds.GetZone(viewName, origin)
	if result != domaintree.ExactMatch {
		return nil, view.ErrNoAuthUpdate
	}

	if updator, ok := zone.GetUpdator(clientIP, false); ok {
		return updator, nil
	} else {
		return nil, view.ErrNoAuthUpdate
	}
}
