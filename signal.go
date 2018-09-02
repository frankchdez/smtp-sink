package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/log"
)

var signalChannel = make(chan os.Signal, 1)

func sigHandler(d *guerrilla.Daemon, logger log.Logger) {
	// handle SIGHUP for reloading the configuration while running
	signal.Notify(signalChannel,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGUSR1,
	)
	// Keep the daemon busy by waiting for signals to come
	for sig := range signalChannel {
		switch sig {
		case syscall.SIGHUP:
			configPath, _ := getConfigPath()
			d.ReloadConfigFile(configPath)
		case syscall.SIGUSR1:
			d.ReopenLogs()
		case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
			logger.Infof("Shutdown signal caught")
			d.Shutdown()
			logger.Infof("Shutdown completed, exiting.")
			return
		default:
			logger.Infof("Shutdown, unknown signal caught")
			return
		}
	}
}
