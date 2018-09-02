package main

import (
	"os"

	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/log"
)

func main() {
	logger, err := log.GetLogger(log.OutputStderr.String(), log.InfoLevel.String())
	if err != nil {
		logger.WithError(err).Errorf("Failed creating a logger to %s", log.OutputStderr)
	}

	d := guerrilla.Daemon{Logger: logger}
	d.AddProcessor("FileExtract", decorator)

	if err = readConfig(&d); err != nil {
		logger.WithError(err).Fatal("Error while reading config")
	}

	// Check that max clients is not greater than system open file limit.
	fileLimit := getFileLimit()
	if fileLimit > 0 {
		maxClients := 0
		for _, s := range d.Config.Servers {
			maxClients += s.MaxClients
		}
		if maxClients > fileLimit {
			logger.Fatalf("Combined max clients for all servers (%d) is greater than open file limit (%d). "+
				"Please increase your open file limit or decrease max clients.", maxClients, fileLimit)
		}
	}

	if err = d.Start(); err != nil {
		logger.WithError(err).Error("Error(s) when starting server(s)")
		os.Exit(1)
	}

	sigHandler(&d, logger)
}
