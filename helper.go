package main

import (
	"bytes"
	"net"
	"text/template"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/sirupsen/logrus"
)

func Sysctl() {
	logrus.Info("[sysctl] enable net.ipv4.ip_forward...")
	if err := sysctl.Set("net.ipv4.ip_forward", "1"); err != nil {
		logrus.Fatalf("[sysctl] failed to set net.ipv4.ip_forward: %v", err)
	}

	logrus.Info("[sysctl] enable net.ipv4.conf.all.route_localnet...")
	if err := sysctl.Set("net.ipv4.conf.all.route_localnet", "1"); err != nil {
		logrus.Fatalf("[sysctl] failed to set net.ipv4.conf.all.route_localnet: %v", err)
	}
}

func IfName() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("[interface] failed to list network interfaces: %v", err)
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				logrus.Errorf("[interface] failed to get addrs: %v", err)
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

	logrus.Error("[interface] failed to get main interface")
	return ""
}

func AutoFix(c string) string {
	var buf bytes.Buffer
	tpl, err := template.New("").Funcs(template.FuncMap{
		"IfName": IfName,
	}).Parse(c)

	if err != nil {
		logrus.Errorf("[autofix] failed to parse template: %v", err)
		return c
	}

	// Auto inject main network interface name
	if err = tpl.Execute(&buf, nil); err != nil {
		logrus.Errorf("[autofix] failed to execute template: %v", err)
		return c
	}

	return buf.String()
}
