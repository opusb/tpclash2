package main

import (
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

	return &tunMode{
		ins:  ip4,
		tpcc: tpcc,
		cc:   cc,
	}, nil
}
