package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-example/exporter"
	"github.com/go-example/utils"
	"go.uber.org/zap/zapcore"
)

func main() {

	router := gin.Default()

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

	//exporter.Start(router, "/metrics", 40, false)
	err = exporter.Start(router,
		exporter.SetMetricsPath("/metrics"),
		exporter.SetMaxRequests(40),
		exporter.SetIncludeExporterMetrics(true),
		exporter.SetLogger(log),
	)
	if err != nil {
		fmt.Println("start exporter error:", err)
		return
	}

	router.Run(":9100")
}
