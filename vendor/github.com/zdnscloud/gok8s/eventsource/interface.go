package eventsource

type EventSource interface {
	GetEventChannel() (<-chan interface{}, error)
}
