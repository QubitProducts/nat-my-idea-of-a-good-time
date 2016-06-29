package main

import (
	"fmt"
	"net"
	"time"

	"testing"
)

func getAddr(host string) (net.Addr, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			return &net.IPAddr{IP: ip}, nil
		}
	}
	return nil, fmt.Errorf("no A or AAAA record")
}

func TestPingGoogle(t *testing.T) {
	addr, err := getAddr("google.com")
	if err != nil {
		t.Error(err)
	}

	err = doPing(addr, time.Second*1)
	if err != nil {
		t.Error(err)
	}
}
