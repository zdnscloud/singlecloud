package predicate

import (
	"github.com/zdnscloud/gok8s/event"
)

type Predicate interface {
	IgnoreCreate(event.CreateEvent) bool
	IgnoreDelete(event.DeleteEvent) bool
	IgnoreUpdate(event.UpdateEvent) bool
	IgnoreGeneric(event.GenericEvent) bool
}
