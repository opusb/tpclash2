package main

import (
	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func ensureSysctl() {
	logrus.Info("[sysctl] enable net.ipv4.ip_forward...")
	val, err := sysctl.Get("net.ipv4.ip_forward")
	if err != nil {
		logrus.Fatalf("failed to read net.ipv4.ip_forward: ", err)
	}
	if val != "1" {
		err := sysctl.Set("net.ipv4.ip_forward", "1")
		if err != nil {
			logrus.Fatalf("failed to set net.ipv4.ip_forward: %v", err)
		}
	}


	logrus.Info("[sysctl] enable net.ipv4.conf.all.route_localnet...")
	val, err = sysctl.Get("net.ipv4.conf.all.route_localnet")
	if err != nil {
		logrus.Fatalf("failed to read net.ipv4.conf.all.route_localnet: ", err)
	}
	if val != "1" {
		err = sysctl.Set("net.ipv4.conf.all.route_localnet", "1")
		if err != nil {
			logrus.Fatalf("failed to set net.ipv4.conf.all.route_localnet: %v", err)
		}
	}
}
