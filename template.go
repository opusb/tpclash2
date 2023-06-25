package main

import (
	"net"
	"os"
	"regexp"
	"text/template"

	"github.com/sirupsen/logrus"
)

var confFuncsMap = template.FuncMap{
	"IfName":     tplIfName,
	"DefaultDNS": tplDefaultDNS,
}

func tplIfName() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("[helper/if-name] failed to list network interfaces: %v", err)
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				logrus.Errorf("[helper/if-name] failed to get addrs: %v", err)
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

	logrus.Error("[helper/if-name] failed to get main interface")
	return ""
}

func tplDefaultDNS() []string {
	resolvConf := "/run/systemd/resolve/resolv.conf"
	_, err := os.Stat(resolvConf)
	if err != nil {
		resolvConf = "/etc/resolv.conf"
	}

	bs, err := os.ReadFile(resolvConf)
	if err != nil {
		logrus.Errorf("[helper/default-dns] failed to read resole.conf: %v", err)
		return nil
	}

	regx := regexp.MustCompile(`(?m)^nameserver\s+(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	matches := regx.FindAllStringSubmatch(string(bs), -1)
	if len(matches) == 0 {
		logrus.Errorf("[helper/default-dns] failed to parse resole.conf: missing nameservers")
		return nil
	}
	var servers []string
	for _, match := range matches {
		servers = append(servers, match[1])
	}
	return servers
}
