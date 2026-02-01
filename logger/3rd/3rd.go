package _rd

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
	MaxLevel
)

type LoggerFunc func(level Level, format string, args ...interface{})

var logger LoggerFunc

func SetLogger(l LoggerFunc) {
	logger = l
}

func New3rdResource(name string) error {
	logger(InfoLevel, "New3rdResource: %s", name)
	logger(DebugLevel, "New3rdResource: %s", name)
	logger(ErrorLevel, "New3rdResource: %s", name)
	logger(FatalLevel, "New3rdResource: %s", name)
	logger(PanicLevel, "New3rdResource: %s", name)
	logger(WarnLevel, "New3rdResource: %s", name)

	return nil
}
