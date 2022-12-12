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

	addMisc() error
	delMisc() error

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

func NewProxyMode(cc *ClashConf, tpcc *TPClashConf) (ProxyMode, error) {
	ip4, err := newIPTables(iptables.ProtocolIPv4)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(conf.ProxyMode) {
	case "tproxy":
		return &tproxyMode{
			ins:  ip4,
			tpcc: tpcc,
			cc:   cc,
		}, nil
	case "tun":
		return &tunMode{
			ins:  ip4,
			tpcc: tpcc,
			cc:   cc,
		}, nil
	}

	return nil, fmt.Errorf("unsupported proxy mode: %s", conf.ProxyMode)
}
