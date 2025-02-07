package public

import (
	"syscall"

	"go.uber.org/zap"
)

// SetLimit Increase resources limitations
func SetLimit() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}

	Logger.Info("SetLimit", zap.Uint64("rLimit.Cur", rLimit.Cur))
}
