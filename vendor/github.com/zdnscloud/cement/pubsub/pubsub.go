package pubsub

type operation int

const (
	sub operation = iota
	pub
	tryPub
	unsub
	unsubAll
	closeTopic
	shutdown
)

// PubSub is a collection of topics.
type PubSub struct {
	cmdChan  chan cmd
	capacity int
}

type cmd struct {
	op     operation
	topics []string
	ch     chan interface{}
	msg    interface{}
}

// New creates a new PubSub and starts a goroutine for handling operations.
// The capacity of the channels created by Sub and SubOnce will be as specified.
func New(capacity int) *PubSub {
	ps := &PubSub{make(chan cmd), capacity}
	go ps.start()
	return ps
}

// Sub returns a channel on which messages published on any of
// the specified topics can be received.
func (ps *PubSub) Sub(topics ...string) chan interface{} {
	ch := make(chan interface{}, ps.capacity)
	ps.AddSub(ch, topics...)
	return ch
}

// AddSub adds subscriptions to an existing channel.
func (ps *PubSub) AddSub(ch chan interface{}, topics ...string) {
	if len(topics) == 0 {
		panic("sub to empty topics")
	}
	ps.cmdChan <- cmd{op: sub, topics: topics, ch: ch}
}

// Pub publishes the given message to all subscribers of
// the specified topics.
func (ps *PubSub) Pub(msg interface{}, topics ...string) {
	ps.cmdChan <- cmd{op: pub, topics: topics, msg: msg}
}

// TryPub publishes the given message to all subscribers of
// the specified topics if the topic has buffer space.
func (ps *PubSub) TryPub(msg interface{}, topics ...string) {
	ps.cmdChan <- cmd{op: tryPub, topics: topics, msg: msg}
}

// Unsub unsubscribes the given channel from the specified
// topics. If no topic is specified, it is unsubscribed
// from all topics.
//
// Unsub must be called from a goroutine that is different from the subscriber.
// The subscriber must consume messages from the channel until it reaches the
// end. Not doing so can result in a deadlock.
func (ps *PubSub) Unsub(ch chan interface{}, topics ...string) {
	if len(topics) == 0 {
		ps.cmdChan <- cmd{op: unsubAll, ch: ch}
		return
	}

	ps.cmdChan <- cmd{op: unsub, topics: topics, ch: ch}
}

// Close closes all channels currently subscribed to the specified topics.
// If a channel is subscribed to multiple topics, some of which is
// not specified, it is not closed.
func (ps *PubSub) Close(topics ...string) {
	ps.cmdChan <- cmd{op: closeTopic, topics: topics}
}

// Shutdown closes all subscribed channels and terminates the goroutine.
func (ps *PubSub) Shutdown() {
	ps.cmdChan <- cmd{op: shutdown}
}

func (ps *PubSub) start() {
	reg := registry{
		topics:    make(map[string][]chan interface{}),
		revTopics: make(map[chan interface{}]map[string]struct{}),
	}

	for cmd := range ps.cmdChan {
		if cmd.topics == nil {
			switch cmd.op {
			case unsubAll:
				reg.removeChannel(cmd.ch)

			case shutdown:
				for ch, _ := range reg.revTopics {
					close(ch)
				}
				return
			}
			continue
		}

		for _, topic := range cmd.topics {
			switch cmd.op {
			case sub:
				reg.add(topic, cmd.ch)

			case tryPub:
				reg.sendNoWait(topic, cmd.msg)

			case pub:
				reg.send(topic, cmd.msg)

			case unsub:
				reg.remove(topic, cmd.ch)

			case closeTopic:
				reg.removeTopic(topic)
			}
		}
	}
}

// registry maintains the current subscription state. It's not
// safe to access a registry from multiple goroutines simultaneously.
type registry struct {
	topics    map[string][]chan interface{}
	revTopics map[chan interface{}]map[string]struct{}
}

func (reg *registry) add(topic string, ch chan interface{}) {
	chs := reg.topics[topic]
	chs = append(chs, ch)
	reg.topics[topic] = chs

	if reg.revTopics[ch] == nil {
		reg.revTopics[ch] = make(map[string]struct{})
	}
	reg.revTopics[ch][topic] = struct{}{}
}

func (reg *registry) send(topic string, msg interface{}) {
	for _, ch := range reg.topics[topic] {
		ch <- msg
	}
}

func (reg *registry) sendNoWait(topic string, msg interface{}) {
	for _, ch := range reg.topics[topic] {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (reg *registry) removeTopic(topic string) {
	for _, ch := range reg.topics[topic] {
		reg.remove(topic, ch)
	}
}

func (reg *registry) removeChannel(ch chan interface{}) {
	for topic := range reg.revTopics[ch] {
		reg.remove(topic, ch)
	}
}

func (reg *registry) remove(topic string, ch chan interface{}) {
	if _, ok := reg.topics[topic]; !ok {
		return
	}

	chs, ok := reg.topics[topic]
	if !ok {
		return
	}

	hasCh := false
	for i, c := range chs {
		if c == ch {
			reg.topics[topic] = append(chs[:i], chs[i+1:]...)
			hasCh = true
			break
		}
	}
	if hasCh == false {
		return
	}

	delete(reg.revTopics[ch], topic)

	if len(reg.topics[topic]) == 0 {
		delete(reg.topics, topic)
	}

	if len(reg.revTopics[ch]) == 0 {
		close(ch)
		delete(reg.revTopics, ch)
	}
}
