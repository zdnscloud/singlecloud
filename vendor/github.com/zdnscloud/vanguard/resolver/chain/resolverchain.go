package chain

import (
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
)

type Resolver interface {
	Resolve(*core.Client)
	Next() Resolver
	SetNext(Resolver)
	ReloadConfig(*config.VanguardConf)
}

type DefaultResolver struct {
	next Resolver
}

func (h *DefaultResolver) Resolve(client *core.Client) {}
func (h *DefaultResolver) Next() Resolver              { return h.next }
func (h *DefaultResolver) SetNext(next Resolver)       { h.next = next }

func BuildResolverChain(resolvers ...Resolver) {
	var prev Resolver
	for i, resolver := range resolvers {
		if i > 0 {
			prev.SetNext(resolver)
		}
		prev = resolver
	}
}

func PassToNext(h Resolver, client *core.Client) {
	if next := h.Next(); next != nil {
		next.Resolve(client)
	}
}

func ReconfigChain(h Resolver, conf *config.VanguardConf) {
	h.ReloadConfig(conf)
	if next := h.Next(); next != nil {
		ReconfigChain(next, conf)
	}
}
