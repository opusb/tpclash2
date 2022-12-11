package main

import (
	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func ensureSysctl() {
	logrus.Info("[sysctl] enable net.ipv4.ip_forward...")
	err := sysctl.Set("net.ipv4.ip_forward", "1")
	if err != nil {
		logrus.Fatalf("failed to set net.ipv4.ip_forward: %v", err)
	}

	logrus.Info("[sysctl] enable net.ipv4.conf.all.route_localnet...")
	err = sysctl.Set("net.ipv4.conf.all.route_localnet", "1")
	if err != nil {
		logrus.Fatalf("failed to set net.ipv4.conf.all.route_localnet: %v", err)
	}
}
