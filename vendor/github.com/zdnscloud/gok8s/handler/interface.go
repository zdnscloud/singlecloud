package handler

import (
	"time"

	"github.com/zdnscloud/gok8s/event"
)

type Result struct {
	Requeue      bool
	RequeueAfter time.Duration
}

type EventHandler interface {
	OnCreate(event.CreateEvent) (Result, error)
	OnUpdate(event.UpdateEvent) (Result, error)
	OnDelete(event.DeleteEvent) (Result, error)
	OnGeneric(event.GenericEvent) (Result, error)
}
