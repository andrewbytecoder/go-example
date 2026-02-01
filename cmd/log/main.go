package main

import (
	"fmt"

	"github.com/go-example/logger"
	_rd "github.com/go-example/logger/3rd"
	"github.com/go-example/utils"
	"go.uber.org/zap/zapcore"
)

func main() {

	log, err := utils.CreateProductZapLogger(
		utils.SetConsoleWriterSyncer(true),
		utils.SetLogFilename("log/exporter.log"),
		utils.SetLogLevel(zapcore.InfoLevel),
		utils.SetLogLevelKey("exporter"),
	)
	if err != nil {
		fmt.Println("create logger error:", err)
		return
	}

	logger.CreateWrapperLogger(log)

	err = _rd.New3rdResource("resource")
	if err != nil {
		return
	}

}
