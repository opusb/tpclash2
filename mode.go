package main

import (
	"github.com/coreos/go-iptables/iptables"
)

type ProxyMode interface {
	EnableProxy() error
	DisableProxy() error
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
