package public

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(debugMode bool) {
	lv := zapcore.InfoLevel
	if debugMode {
		lv = zapcore.DebugLevel
	}

	// create logger with sampler
	encCfg := zap.NewDevelopmentEncoderConfig()
	enc := zapcore.NewConsoleEncoder(encCfg)
	core := zapcore.NewCore(enc, zapcore.AddSync(zapcore.Lock(os.Stderr)), lv)
	sampler := zapcore.NewSamplerWithOptions(core, time.Second, 2, 50)
	Logger = zap.New(sampler, zap.Development(), zap.AddCaller())
}
