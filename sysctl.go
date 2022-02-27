package main

import (
	"fmt"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func applySysctl() error {
	logrus.Info("[sysctl] enable net.ipv4.ip_forward...")
	err := sysctl.Set("net.ipv4.ip_forward", "1")
	if err != nil {
		return fmt.Errorf("failed to set net.ipv4.ip_forward: %w", err)
	}

	logrus.Info("[sysctl] net.ipv4.conf.all.route_localnet...")
	err = sysctl.Set("net.ipv4.conf.all.route_localnet", "1")
	if err != nil {
		return fmt.Errorf("failed to set net.ipv4.conf.all.route_localnet: %w", err)
	}

	return nil
}
