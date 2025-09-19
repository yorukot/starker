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
	LogTypeLog      LogType = "log"
	LogTypeProgress LogType = "progress"
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

func (sc StreamChan) LogLog(message string) {
	sc.LogChan <- LogMessage{
		Type:    LogTypeLog,
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

func LogLog(message string) LogMessage {
	return LogMessage{
		Type:    LogTypeLog,
		Message: message,
	}
}


func NewStreamChan() StreamChan {
	return StreamChan{
		LogChan:      make(chan LogMessage, 100),
		ErrChan:      make(chan LogMessage, 100),
		ProgressChan: make(chan LogMessage, 100),
		FinalError:   make(chan error, 1),
		DoneChan:     make(chan bool, 1),
	}
}
