package alarm

const (
	UnackNumber MessageType = "UnackNumber"
	UnackAlarm  MessageType = "UnackAlarm"
)

type MessageType string

type Message struct {
	Type    MessageType
	Payload interface{}
}
