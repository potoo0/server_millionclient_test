package public

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(debugMode bool) {
	var err error
	lv := zapcore.InfoLevel
	if debugMode {
		lv = zapcore.DebugLevel
	}
	if Logger, err = zap.NewDevelopment(zap.IncreaseLevel(lv)); err != nil {
		panic(err)
	}
}
