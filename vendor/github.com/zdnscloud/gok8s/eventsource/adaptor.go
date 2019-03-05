package eventsource

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/zdnscloud/gok8s/event"
)

var _ cache.ResourceEventHandler = &HandlerAdaptor{}

type HandlerAdaptor struct {
	ch chan<- interface{}
}

func newHandlerAdaptor(ch chan<- interface{}) *HandlerAdaptor {
	return &HandlerAdaptor{
		ch: ch,
	}
}

func (h *HandlerAdaptor) OnAdd(obj interface{}) {
	c := event.CreateEvent{}

	if o, err := meta.Accessor(obj); err == nil {
		c.Meta = o
	} else {
		return
	}

	if o, ok := obj.(runtime.Object); ok {
		c.Object = o
	} else {
		return
	}

	h.ch <- c
}

func (h *HandlerAdaptor) OnUpdate(oldObj, newObj interface{}) {
	u := event.UpdateEvent{}

	if o, err := meta.Accessor(oldObj); err == nil {
		u.MetaOld = o
	} else {
		return
	}

	if o, ok := oldObj.(runtime.Object); ok {
		u.ObjectOld = o
	} else {
		return
	}

	if o, err := meta.Accessor(newObj); err == nil {
		u.MetaNew = o
	} else {
		return
	}

	if o, ok := newObj.(runtime.Object); ok {
		u.ObjectNew = o
	} else {
		return
	}

	h.ch <- u
}

func (h *HandlerAdaptor) OnDelete(obj interface{}) {
	d := event.DeleteEvent{}

	// Deal with tombstone events by pulling the object out.  Tombstone events wrap the object in a
	// DeleteFinalStateUnknown struct, so the object needs to be pulled out.
	// Copied from sample-controller
	// This should never happen if we aren't missing events, which we have concluded that we are not
	// and made decisions off of this belief.  Maybe this shouldn't be here?
	var ok bool
	if _, ok = obj.(metav1.Object); !ok {
		// If the object doesn't have Metadata, assume it is a tombstone object of type DeletedFinalStateUnknown
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}

		obj = tombstone.Obj
	}

	if o, err := meta.Accessor(obj); err == nil {
		d.Meta = o
	} else {
		return
	}

	if o, ok := obj.(runtime.Object); ok {
		d.Object = o
	} else {
		return
	}

	h.ch <- d
}
