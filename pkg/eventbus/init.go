package eventbus

import (
	"github.com/zdnscloud/cement/pubsub"
)

const EventBufLen = 1000

var eventBus *pubsub.PubSub

func GetEventBus() *pubsub.PubSub {
	return eventBus
}

func init() {
	eventBus = pubsub.New(EventBufLen)
}
