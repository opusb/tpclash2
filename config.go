package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type TPClashConf struct {
	ClashHome   string
	ClashConfig string
	ClashUI     string

	ClashUser      string
	HijackIP       []net.IP
	DisableExtract bool
	AutoExit       bool

	Debug bool
}

type ClashConf struct {
	Debug         bool
	InterfaceName string
}

// ParseClashConf Parses clash configuration and performs necessary checks
// based on proxy mode
func ParseClashConf() (*ClashConf, error) {
	debug := viper.GetString("log-level")
	enhancedMode := viper.GetString("dns.enhanced-mode")
	dnsListen := viper.GetString("dns.listen")
	fakeIPRange := viper.GetString("dns.fake-ip-range")
	interfaceName := viper.GetString("interface-name")
	tunEnabled := viper.GetBool("tun.enable")
	tunAutoRoute := viper.GetBool("tun.auto-route")
	metaIPtables := viper.GetBool("iptables.enable")

	// common check
	if strings.ToLower(enhancedMode) != "fake-ip" {
		return nil, fmt.Errorf("only support fake-ip dns mode(dns.enhanced-mode)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(dnsListen)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config(dns.listen): %v", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config(dns.listen): %v", err)
	}
	if dport < 1 {
		return nil, fmt.Errorf("dns port in clash config is missing(dns.listen)")
	}

	dhost := net.ParseIP(dnsHost)
	if dhost == nil {
		return nil, fmt.Errorf("dns listening address parse failed(dns.listen): is not a valid IP address")
	}

	if interfaceName == "" {
		return nil, fmt.Errorf("failed to parse clash interface name(interface-name): interface-name must be set")
	}

	if fakeIPRange == "" {
		return nil, fmt.Errorf("failed to parse clash fake ip range name(dns.fake-ip-range): fake-ip-range must be set")
	}

	if !tunEnabled {
		return nil, fmt.Errorf("tun must be enabled in tun mode(tun.enable)")
	}

	if !tunAutoRoute {
		return nil, fmt.Errorf("must be enabled auto-route in tun mode(tun.auto-route)")
	}

	if metaIPtables {
		return nil, fmt.Errorf("meta kernel must turn off iptables(iptables.enable)")
	}

	return &ClashConf{
		Debug:         strings.ToLower(debug) == "debug",
		InterfaceName: interfaceName,
	}, nil
}
