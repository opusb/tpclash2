package main

import (
	"bytes"
	"context"

	"github.com/sirupsen/logrus"

	arp "github.com/irai/packet/handlers/arp_spoofer"

	"github.com/irai/packet"
)

type ARPHijacker struct {
	tpcc *TPClashConf
	cc   *ClashConf
	ips  map[string]struct{}
}

func NewARPHijacker(cc *ClashConf, tpcc *TPClashConf) *ARPHijacker {
	ips := map[string]struct{}{}
	for _, ip := range tpcc.HijackIP {
		ips[ip.String()] = struct{}{}
	}

	return &ARPHijacker{
		tpcc: tpcc,
		cc:   cc,
		ips:  ips,
	}
}

func (h *ARPHijacker) hijack(ctx context.Context) error {
	s, err := packet.NewSession(h.cc.InterfaceName)
	if err != nil {
		return err
	}

	sf, err := arp.New(s)
	if err != nil {
		return err
	}

	// start packet processing goroutine
	go func() {
		defer s.Close()
		defer func() { _ = sf.Close() }()
		buffer := make([]byte, packet.EthMaxSize)

		for {
			n, _, err := s.ReadFrom(buffer)
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					logrus.Errorf("failed to reading packet: %v", err)
				}
				return
			}

			// Ignore packets sent by us
			if bytes.Equal(packet.SrcMAC(buffer[:n]), s.NICInfo.HostAddr4.MAC) {
				continue
			}

			// memory map packet so we can access all fields
			frame, err := s.Parse(buffer[:n])
			if err != nil {
				logrus.Debugf("[ARP] failed to parse frame: %v", err)
				continue
			}

			s.Notify(frame)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case notification := <-s.C:
				_, all := h.ips["0.0.0.0"]
				_, contain := h.ips[notification.Addr.IP.String()]
				if !all && !contain {
					continue
				}

				switch notification.Online {
				case true:
					logrus.Infof("[ARP] %s is online...", notification)
					_, _ = sf.StartHunt(notification.Addr)
				default:
					logrus.Warnf("[ARP] %s is offline...", notification)
					_, _ = sf.StopHunt(notification.Addr)
				}
			}
		}
	}()

	return sf.Scan()
}
