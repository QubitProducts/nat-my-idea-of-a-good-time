package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	checkTarget           string
	checkTimeout          time.Duration
	checkInterval         time.Duration
	checkFailureThreshold int

	prometheusAddress string

	dryRun bool
)

func init() {
	flag.StringVar(&checkTarget, "target", os.Getenv("NAT_TARGET", ""), "Hostname to test")
	flag.DurationVar(&checkTimeout, "timeout", time.Millisecond*mustInt(os.Getenv("NAT_TIMEOUT_MS", "500")), "Timeout for NAT check in milliseconds")
	flag.DurationVar(&checkInterval, "interval", time.Milliseconds*mustInt(os.Getenv("NAT_INTERVAL_MS", "1000")), "Interval to test connectivity in milliseconds")
	flag.IntVar(&checkFailureThreshold, "threshold", mustInt(os.Getenv("NAT_THRESHOLD", "5")), "Number of times the check may fail before action is taken")

	flag.StringVar(&prometheusAddress, "prometheus", os.Getenv("NAT_PROMETHEUS", ":8080"), "Address to expose the Prometheus monitoring handler")

	flag.BoolVar(&dryRun, "dry-run", mustBool(os.Getenv("NAT_DRY_RUN"), "true"), "Prevents any side affects occuring")
}

func mustInt(str string) int {
	v, err := strconv.Atoi(str)
	if err != nil {
		glog.Fatalf("Failed to parse %v as integer: %v", str, err)
	}
	return v
}

func main() {
	flag.Parse()

	if checkTarget == "" {
		glog.Fatalln("No health check target specified")
	}

	fa := newFanoutAction()
	fa.AddAction("routetable", makeRouteTableFailoverAction())
	fa.AddAction("email", makeEmailAction())

	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})
	go http.ListenAndServe(prometheusAddress, nil)

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
				go action.Trigger(err)
				consecutiveFailures = 0
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

		err = doPing(addr, checkTimeout*3/2)
		if err != nil {
			glog.Errorf("Failed to ping %v: %v", host, err)
		}
		res <- err
	}()

	return res
}
