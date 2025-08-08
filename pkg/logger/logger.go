package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/yorukot/stargo/internal/config"
)

// InitLogger initialize the logger
func InitLogger() {
	appEnv := os.Getenv("APP_ENV")

	var logger *zap.Logger

	if appEnv == string(config.AppEnvDev) {
		devConfig := zap.NewDevelopmentConfig()
		devConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, _ = devConfig.Build()
	} else {
		logger = zap.Must(zap.NewProduction())
	}

	zap.ReplaceGlobals(logger)

	zap.L().Info("Logger initialized")

	defer logger.Sync()
}
