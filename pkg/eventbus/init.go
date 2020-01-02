package eventbus

import (
	"github.com/zdnscloud/cement/pubsub"
)

const EventBufLen = 1000

var EventBus *pubsub.PubSub

func init() {
	EventBus = pubsub.New(EventBufLen)
}
