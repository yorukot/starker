package core

type ProgressDetail struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

type ProgressMessage struct {
	Status         string         `json:"status"`
	ProgressDetail ProgressDetail `json:"progressDetail"`
	Progress       string         `json:"progress"`
	ID             string         `json:"id"`
}

type StreamChan struct {
	LogChan      chan LogMessage
	ErrChan      chan LogMessage
	ProgressChan chan LogMessage
	FinalError   chan error
	DoneChan     chan bool
}

type LogType string

const (
	LogTypeError    LogType = "error"
	LogTypeInfo     LogType = "info"
	LogTypeProgress LogType = "progress"
	LogTypeStep     LogType = "step"
)

type LogMessage struct {
	Message string  `json:"message"`
	Data    any     `json:"data,omitempty"`
	Type    LogType `json:"type"`
}

func (sc StreamChan) LogError(message string) {
	sc.ErrChan <- LogMessage{
		Type:    LogTypeError,
		Message: message,
	}
}

func (sc StreamChan) LogInfo(message string) {
	sc.LogChan <- LogMessage{
		Type:    LogTypeInfo,
		Message: message,
	}
}

func (sc StreamChan) LogStep(message string) {
	sc.LogChan <- LogMessage{
		Type:    LogTypeStep,
		Message: message,
	}
}

func (sc StreamChan) LogProgress(progress ProgressMessage) {
	sc.ProgressChan <- LogMessage{
		Type: LogTypeProgress,
		Data: progress,
	}
}

func LogError(message string) LogMessage {
	return LogMessage{
		Type:    LogTypeError,
		Message: message,
	}
}

func LogInfo(message string) LogMessage {
	return LogMessage{
		Type:    LogTypeInfo,
		Message: message,
	}
}

func LogStep(message string) LogMessage {
	return LogMessage{
		Type:    LogTypeStep,
		Message: message,
	}
}

func NewStreamChan() StreamChan {
	return StreamChan{
		LogChan:    make(chan LogMessage, 100),
		ErrChan:    make(chan LogMessage, 100),
		FinalError: make(chan error, 1),
		DoneChan:   make(chan bool, 1),
	}
}
