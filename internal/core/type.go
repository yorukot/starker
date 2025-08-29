package core

type StreamChan struct {
	LogChan    chan LogMessage
	ErrChan    chan LogMessage
	FinalError chan error
	DoneChan   chan bool
}

type LogType string

const (
	LogTypeError LogType = "error"
	LogTypeInfo  LogType = "info"
	LogTypeStep  LogType = "step"
)

type LogMessage struct {
	Message string  `json:"message"`
	Type    LogType `json:"type"`
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
