package eventbus

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/gorest/resource"
)

type MyResource struct {
	resource.ResourceBase
}

func TestPubSubResource(t *testing.T) {
	ch := SubscribeResourceEvent(MyResource{})
	createEventCount := 0
	deleteEventCount := 0
	updateEventCount := 0

	sendCreateEventCount := 10
	sendDeleteEventCount := 20
	sendUpdateEventCount := 30
	for i := 0; i < sendCreateEventCount; i++ {
		PublishResourceCreateEvent(&MyResource{})
	}

	for i := 0; i < sendDeleteEventCount; i++ {
		PublishResourceDeleteEvent(&MyResource{})
	}

	for i := 0; i < sendUpdateEventCount; i++ {
		PublishResourceUpdateEvent(&MyResource{}, &MyResource{})
	}

	Shutdown()

	for e := range ch {
		switch e.(type) {
		case ResourceCreateEvent:
			createEventCount += 1
		case ResourceDeleteEvent:
			deleteEventCount += 1
		case ResourceUpdateEvent:
			updateEventCount += 1
		}
	}

	ut.Equal(t, createEventCount, sendCreateEventCount)
	ut.Equal(t, deleteEventCount, sendDeleteEventCount)
	ut.Equal(t, updateEventCount, sendUpdateEventCount)
}
