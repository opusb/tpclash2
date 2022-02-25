package main

import (
	"context"
	"github.com/Dreamacro/clash/config"
	"github.com/Dreamacro/clash/hub"
	"github.com/sirupsen/logrus"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	clsconst "github.com/Dreamacro/clash/constant"
	"go.uber.org/automaxprocs/maxprocs"
)

func run() {
	logrus.Info("starting clash...")

	// support container
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))

	// local config
	clsconst.SetHomeDir(clashHome)
	clsconst.SetConfig(clashConfig)

	if err := config.Init(clashHome); err != nil {
		logrus.Fatal("Initial configuration directory error: %s", err.Error())
	}

	// start clash
	if err := hub.Parse(); err != nil {
		logrus.Fatal("Parse config error: %s", err.Error())
	}

	logrus.Info("starting clash dashboard...")
	go func() {
		var uiHandler http.Handler
		switch strings.ToLower(clashUI) {
		case "yacd":
			uiHandler = http.FileServer(http.FS(yacd))
		default:
			uiHandler = http.FileServer(http.FS(official))
		}
		_ = http.ListenAndServe(clashUIAddr, uiHandler)
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()
}

func fix() {
	if err := fixSysctl(); err != nil {
		logrus.Fatalf("Fix Sysctl Error: %s", err)
	}

	if err := fixRoute(); err != nil {
		logrus.Fatalf("Fix IP Route Error: %s", err)
	}

	if err := fixIPTables(); err != nil {
		logrus.Fatalf("Fix IPTables Error: %s", err)
	}
}
