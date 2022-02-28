package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

func run() {
	logrus.Info("starting clash...")
	go func() {
		cmd := exec.Command(filepath.Join(clashHome, "xclash"), "-c", clashConfig, "-d", clashHome, "-ext-ui", clashUI)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		_ = cmd.Run()
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()
	logrus.Info("TPClash exit...")
}

func fix() {
	if err := applySysctl(); err != nil {
		logrus.Fatalf("Fix Sysctl Error: %s", err)
	}

	if err := applyRoute(); err != nil {
		logrus.Fatalf("Fix IP Route Error: %s", err)
	}

	if err := applyIPTables(); err != nil {
		logrus.Fatalf("Fix IPTables Error: %s", err)
	}
}

func clean() {
	if err := cleanIPTables(); err != nil {
		logrus.Fatalf("Clean IPTables Error: %v", err)
	}

	if err := cleanRoute(); err != nil {
		logrus.Fatalf("Clean Route Error: %v", err)
	}
}
