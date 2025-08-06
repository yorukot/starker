package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger initialize the logger
func InitLogger() {
	logLevel := os.Getenv("APP_ENV")

	var logger *zap.Logger

	if logLevel == "dev" {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, _ = config.Build()
	} else {
		logger = zap.Must(zap.NewProduction())
	}

	zap.ReplaceGlobals(logger)

	zap.L().Info("Logger initialized")

	defer logger.Sync()
}
