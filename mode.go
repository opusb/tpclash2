package main

import (
	"fmt"
	"strings"

	"github.com/coreos/go-iptables/iptables"
)

type ProxyMode interface {
	addForward() error
	delForward() error

	addForwardDNS() error
	delForwardDNS() error

	addLocal() error
	delLocal() error

	addLocalDNS() error
	delLocalDNS() error

	apply() error
	clean() error

	EnableForward() error
	DisableForward() error
}

func process(fns ...func() error) error {
	var err error
	for _, fn := range fns {
		if err = fn(); err != nil {
			return err
		}
	}
	return nil
}

func parseProxyMode() error {
	ip4, err := newIPTables(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}

	switch strings.ToLower(conf.ProxyMode) {
	case "tproxy":
		proxyMode = &tproxyMode{
			ins:  ip4,
			tpcc: &conf,
			cc:   &clashConf,
		}
		return nil
	case "ebpf":
		proxyMode = &ebpfMode{
			ins:  ip4,
			tpcc: &conf,
			cc:   &clashConf,
		}
		return nil
	}

	return fmt.Errorf("unsupported proxy mode: %s", conf.ProxyMode)
}
