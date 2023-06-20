package main

import (
	"github.com/google/nftables"
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

	conn, err := nftables.New()
	if err != nil {
		return nil, err
	}

	return &tunMode{
		nft:  conn,
		tpcc: tpcc,
		cc:   cc,
	}, nil
}
