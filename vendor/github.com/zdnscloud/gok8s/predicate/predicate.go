package predicate

import (
	"github.com/zdnscloud/gok8s/event"
)

type funcs struct {
	IgnoreCreateFunc  func(event.CreateEvent) bool
	IgnoreDeleteFunc  func(event.DeleteEvent) bool
	IgnoreUpdateFunc  func(event.UpdateEvent) bool
	IgnoreGenericFunc func(event.GenericEvent) bool
}

func NewDefaultPredicate() Predicate {
	return funcs{}
}

func (p funcs) IgnoreCreate(e event.CreateEvent) bool {
	if p.IgnoreCreateFunc != nil {
		return p.IgnoreCreateFunc(e)
	}
	return false
}

func (p funcs) IgnoreDelete(e event.DeleteEvent) bool {
	if p.IgnoreDeleteFunc != nil {
		return p.IgnoreDeleteFunc(e)
	}
	return false
}

func (p funcs) IgnoreUpdate(e event.UpdateEvent) bool {
	if p.IgnoreUpdateFunc != nil {
		return p.IgnoreUpdateFunc(e)
	}
	return false
}

func (p funcs) IgnoreGeneric(e event.GenericEvent) bool {
	if p.IgnoreGenericFunc != nil {
		return p.IgnoreGenericFunc(e)
	}
	return false
}

type ignoreUnchangedUpdate struct {
	funcs
}

func NewIgnoreUnchangedUpdate() Predicate {
	return ignoreUnchangedUpdate{}
}

func (ignoreUnchangedUpdate) IgnoreUpdate(e event.UpdateEvent) bool {
	if e.MetaOld == nil ||
		e.ObjectOld == nil ||
		e.ObjectNew == nil ||
		e.MetaNew == nil ||
		e.MetaNew.GetResourceVersion() == e.MetaOld.GetResourceVersion() {
		return true
	}

	return false
}
