package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// parseClashConf Parses clash configuration and performs necessary checks
// based on proxy mode
func parseClashConf() error {
	debug := viper.GetString("log-level")
	enhancedMode := viper.GetString("dns.enhanced-mode")
	dnsListen := viper.GetString("dns.listen")
	tproxyPort := viper.GetInt("tproxy-port")
	fakeIPRange := viper.GetString("dns.fake-ip-range")
	interfaceName := viper.GetString("interface-name")
	tunEnabled := viper.GetBool("tun.enable")
	routingMark := viper.GetInt("routing-mark")

	// common check
	if strings.ToLower(enhancedMode) != "fake-ip" {
		return fmt.Errorf("only support fake-ip dns mode(dns.enhanced-mode)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(dnsListen)
	if err != nil {
		return fmt.Errorf("failed to parse clash dns listen config(dns.listen): %v", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return fmt.Errorf("failed to parse clash dns listen config(dns.listen): %v", err)
	}
	if dport < 1 {
		return fmt.Errorf("dns port in clash config is missing(dns.listen)")
	}

	if interfaceName == "" {
		return fmt.Errorf("failed to parse clash interface name(interface-name): interface-name must be set")
	}

	if fakeIPRange == "" {
		fakeIPRange = "198.18.0.1/16"
	}

	switch conf.ProxyMode {
	case "tproxy":
		if tproxyPort < 1 {
			return fmt.Errorf("tproxy port in clash config is missing(tproxy-port)")
		}
	case "ebpf":
		if tproxyPort > 0 {
			return fmt.Errorf("please delete the tproxy port in ebpf mode(tproxy-port)")
		}
		if !tunEnabled {
			return fmt.Errorf("tun must be enabled in ebpf mode(tun.enable)")
		}
		if routingMark < 1 {
			return fmt.Errorf("routing-mark must be set in ebpf mode(routing-mark)")
		}
	}

	clashConf = ClashConf{
		Debug:         strings.ToLower(debug) == "debug",
		EnhancedMode:  enhancedMode,
		DNSHost:       dnsHost,
		DNSPort:       dnsPort,
		TProxyPort:    strconv.Itoa(tproxyPort),
		FakeIPRange:   fakeIPRange,
		InterfaceName: interfaceName,
	}

	return nil
}
