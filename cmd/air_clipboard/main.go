package main

import (
	"air_clipboard/discovery"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

const (
	DiscoveryPort = 9456
	TransferPort  = 9457
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("new logger failed, err=%s", err))
	}
	defer logger.Sync()
	sugaredLogger := logger.Sugar()

	defer func() {
		if e := recover(); e != nil {
			sugaredLogger.Infof("panic recover, err = %s", e)
		}
	}()

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT)

	discoveryService := discovery.New(sugaredLogger, DiscoveryPort, 5)
	go discoveryService.Start()

	sugaredLogger.Info("air_clipboard running...")

	<-exitChan
	sugaredLogger.Info("air_clipboard exit.")
	os.Exit(0)
}
