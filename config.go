package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/flashmob/go-guerrilla"
)

func getConfigPath() (configPath string, pidFile string) {
	fullexecpath := os.Args[0]

	dir, execname := filepath.Split(fullexecpath)
	ext := filepath.Ext(execname)
	name := execname[:len(execname)-len(ext)]

	return filepath.Join(dir, name+".conf"), "/var/run/" + name + ".pid"
}

func readConfig(d *guerrilla.Daemon) error {
	configPath, pidFile := getConfigPath()
	if _, err := d.LoadConfig(configPath); err != nil {
		return err
	}

	if len(d.Config.PidFile) == 0 {
		d.Config.PidFile = pidFile
	}

	if len(d.Config.AllowedHosts) == 0 {
		return errors.New("Empty `allowed_hosts` is not allowed")
	}
	return nil
}

func getFileLimit() int {
	cmd := exec.Command("ulimit", "-n")
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return -1
	}
	return limit
}
