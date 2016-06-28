package main

import (
	"flag"
	"fmt"
	"time"
	"net"

	"github.com/golang/glog"
)

var (
	checkTarget string
	checkTimeout time.Duration
	checkInterval time.Duration
	checkFailureThreshold int

	dryRun bool
)

func init() {
	flag.StringVar(&checkTarget, "target", "", "Hostname to test")
	flag.DurationVar(&checkTimeout, "check-timeout", time.Millisecond*500, "Timeout for NAT check")
	flag.DurationVar(&checkInterval, "interval", time.Second, "Interval to test connectivity")
	flag.IntVar(&checkFailureThreshold, "threshold", 5, "Number of times the check may fail before action is taken")

	flag.BoolVar(&dryRun, "dry-run", true, "Prevents any side affects occuring")
}

func main() {
	flag.Parse()

	if checkTarget == "" {
		glog.Fatalln("No health check target specified")
	}

	fa := newFanoutAction()
	fa.AddAction(makeRouteTableFailoverAction())
	fa.AddAction(makeEmailAction())

	healthChecker(fa)
}

func healthChecker(action Action) {
	ticker := time.Tick(checkInterval)

	consecutiveFailures := 0
	for range ticker {
		var err error

		checkChan := checkEndpoint(checkTarget)
		select {
		case <-time.After(checkTimeout):
			err = fmt.Errorf("Check timed out after %v", checkTimeout)
			glog.Errorf("Check timed out after %v", checkTimeout)
		case err = <-checkChan:
		}

		if err == nil {
			consecutiveFailures = 0
			glog.Infof("Check succeeded")
		} else {
			consecutiveFailures++
			glog.Errorf("%v consecutive failures", consecutiveFailures)

			if consecutiveFailures >= checkFailureThreshold {
				glog.Errorf("Consecutive failures greater than configured threshold")
				action.Trigger(err)
			}
		}
	}
}

func checkEndpoint(host string) chan error {
	res := make(chan error, 1)

	go func() {
		addr, err := net.ResolveIPAddr("ip4:icmp", host)
		if err != nil {
			glog.Errorf("Failed to resolve %v", host)
			res <- err
			return
		}

		err = doPing(addr, checkTimeout * 3 / 2)
		if err != nil {
			glog.Errorf("Failed to ping %v: %v", host, err)
		}
		res <- err
	}()

	return res
}
