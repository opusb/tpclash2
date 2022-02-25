package main

import (
	"fmt"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func fixSysctl() error {
	logrus.Info("check sysctl...")
	err := sysctl.Set("net.ipv4.ip_forward", "1")
	if err != nil {
		return fmt.Errorf("failed to set net.ipv4.ip_forward: %w", err)
	}
	err = sysctl.Set("net.ipv4.conf.all.route_localnet", "1")
	if err != nil {
		return fmt.Errorf("failed to set net.ipv4.conf.all.route_localnet: %w", err)
	}

	return nil
}
