package middleware

import (
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ZapLoggerMiddleware is a middleware that logs the incoming request and the response time
func ZapLoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()

			start := time.Now()

			next.ServeHTTP(w, r)

			logger.Info(GenerateDiffrentColorForMethod(r.Method)+" request completed",
				zap.String("request_id", requestID),
				zap.String("path", r.URL.Path),
				zap.String("user_agent", r.UserAgent()),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// GenerateDiffrentColorForMethod generate a different color for the method
func GenerateDiffrentColorForMethod(method string) string {
	if os.Getenv("APP_ENV") == "dev" {
		switch method {
		case "GET":
			return color.GreenString(method)
		case "POST":
			return color.BlueString(method)
		case "PUT":
			return color.YellowString(method)
		case "PATCH":
			return color.MagentaString(method)
		case "DELETE":
			return color.RedString(method)
		case "OPTIONS":
			return color.CyanString(method)
		case "HEAD":
			return color.WhiteString(method)
		default:
			return method
		}
	}
	return method
}
