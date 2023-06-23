package main

import (
	"net"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func Sysctl() {
	logrus.Info("[helper] enable net.ipv4.ip_forward...")
	if err := sysctl.Set("net.ipv4.ip_forward", "1"); err != nil {
		logrus.Fatalf("[helper] failed to set net.ipv4.ip_forward: %v", err)
	}

	logrus.Info("[helper] enable net.ipv4.conf.all.route_localnet...")
	if err := sysctl.Set("net.ipv4.conf.all.route_localnet", "1"); err != nil {
		logrus.Fatalf("[helper] failed to set net.ipv4.conf.all.route_localnet: %v", err)
	}
}

func IfName() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("[helper] failed to list network interfaces: %v", err)
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				logrus.Errorf("[helper] failed to get addrs: %v", err)
				return ""
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return iface.Name
					}
				}
			}
		}
	}

	logrus.Error("[helper] failed to get main interface")
	return ""
}
