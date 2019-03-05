package eventsource

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/zdnscloud/gok8s/cache"
)

const (
	defaultBufferSize = 1024
)

type resourceEventSource struct {
	gvk   schema.GroupVersionKind
	cache cache.Cache
}

var _ EventSource = &resourceEventSource{}

func New(gvk schema.GroupVersionKind, cache cache.Cache) EventSource {
	return &resourceEventSource{
		gvk:   gvk,
		cache: cache,
	}
}

func (l *resourceEventSource) GetEventChannel() (<-chan interface{}, error) {
	i, err := l.cache.GetInformerForKind(l.gvk)
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{})
	i.AddEventHandler(newHandlerAdaptor(ch))
	return ch, nil
}

func (l *resourceEventSource) String() string {
	return fmt.Sprintf("kind source: %v", l.gvk.String())
}
