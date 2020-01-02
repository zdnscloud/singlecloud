package alarm

type MessageType string

const (
	UnackNumber MessageType = "UnackNumber"
	UnackAlarm  MessageType = "UnackAlarm"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}
