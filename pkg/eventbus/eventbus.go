package eventbus

import (
	"fmt"

	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gorest/resource"
)

const EventBufLen = 1000

var eventBus *pubsub.PubSub

func init() {
	eventBus = pubsub.New(EventBufLen)
}

type ResourceCreateEvent struct {
	Resource resource.Resource
}

type ResourceDeleteEvent struct {
	Resource resource.Resource
}

type ResourceUpdateEvent struct {
	ResourceOld resource.Resource
	ResourceNew resource.Resource
}

func PublishResourceCreateEvent(r resource.Resource) {
	eventBus.Pub(ResourceCreateEvent{
		Resource: r,
	}, resource.DefaultKindName(r))
}

func PublishResourceDeleteEvent(r resource.Resource) {
	eventBus.Pub(ResourceDeleteEvent{
		Resource: r,
	}, resource.DefaultKindName(r))
}

func PublishResourceUpdateEvent(resourceOld, resourceNew resource.Resource) {
	oldKind := resource.DefaultKindName(resourceOld)
	newKind := resource.DefaultKindName(resourceNew)
	if oldKind != newKind {
		panic(fmt.Sprintf("publish update event with different kind %s:%s", oldKind, newKind))
	}

	eventBus.Pub(ResourceUpdateEvent{
		ResourceOld: resourceOld,
		ResourceNew: resourceNew,
	}, oldKind)
}

func SubscribeResourceEvent(kinds ...resource.ResourceKind) chan interface{} {
	topics := make([]string, 0, len(kinds))
	for _, k := range kinds {
		topics = append(topics, resource.DefaultKindName(k))
	}
	return eventBus.Sub(topics...)
}

func UnsubscribeResourceEvent(ch chan interface{}) {
	eventBus.Unsub(ch)
}

func Shutdown() {
	eventBus.Shutdown()
}
