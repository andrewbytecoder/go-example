package logger

import (
	_rd "github.com/go-example/logger/3rd"
	"go.uber.org/zap"
)

type loggerFunc func(template string, args ...interface{})

type WrapperLogger struct {
	loggers []loggerFunc
}

var w *WrapperLogger

func Logf(level _rd.Level, format string, args ...interface{}) {
	w.loggers[level](format, args...)
}

func CreateWrapperLogger(logger *zap.Logger) {
	// 创建一个wrapper
	wl := &WrapperLogger{
		loggers: make([]loggerFunc, _rd.MaxLevel),
	}
	// 必须跳过一层，否则日志记录会少一层
	sugarLogger := logger.Sugar().WithOptions(zap.AddCallerSkip(1))

	wl.loggers[_rd.DebugLevel] = sugarLogger.Debugf
	wl.loggers[_rd.InfoLevel] = sugarLogger.Infof
	wl.loggers[_rd.WarnLevel] = sugarLogger.Warnf
	wl.loggers[_rd.ErrorLevel] = sugarLogger.Errorf
	wl.loggers[_rd.PanicLevel] = sugarLogger.Panicf
	wl.loggers[_rd.FatalLevel] = sugarLogger.Fatalf

	w = wl

	_rd.SetLogger(Logf)
}
