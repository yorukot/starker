package dockerutils

import (
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
)

type DockerHandler struct {
	Client          *client.Client
	Project         *types.Project
	NamingGenerator *generator.NamingGenerator
	DB              *pgxpool.Pool
	ConnectionPool  *connection.ConnectionPool
	StreamChan      StreamChan
}

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
	LogStep      LogType = "step"
)

type LogMessage struct {
	Message string  `json:"message"`
	Type    LogType `json:"type"`
}

func NewStreamChan() StreamChan {
	return StreamChan{
		LogChan:    make(chan LogMessage, 100),
		ErrChan:    make(chan LogMessage, 100),
		FinalError: make(chan error, 1),
		DoneChan:   make(chan bool, 1),
	}
}

// StreamingResult contains the result of a Docker operation with streaming channels
type StreamingResult struct {
	StreamChan StreamChan
}
