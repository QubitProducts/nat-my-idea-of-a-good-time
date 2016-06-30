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
	subnetName string

	checkTarget           string
	checkTimeout          time.Duration
	checkInterval         time.Duration
	checkFailureThreshold int

	prometheusAddress string

	dryRun bool

	checkDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "natcheck_ping_duration_seconds",
		Help: "The time taken for the check to run, bounded by the check timeout",
		Buckets: prometheus.LinearBuckets(0, 200, 10),
	},
		[]string{"subnet"},
	)
	checkCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "natcheck_ping_total",
		Help: "The number of times that the check has been run, with labels for different outcomes",
	},
		[]string{"subnet", "result"},
	)
)

func init() {
	flag.StringVar(&subnetName, "name", getEnv("NAT_NAME", ""), "Name of the nat/subnet/route table combination is being monitored")
	flag.StringVar(&checkTarget, "target", getEnv("NAT_TARGET", ""), "Hostname to test")
	flag.DurationVar(&checkTimeout, "timeout", getEnvMs("NAT_TIMEOUT_MS", 500), "Timeout for NAT check in milliseconds")
	flag.DurationVar(&checkInterval, "interval", getEnvMs("NAT_INTERVAL_MS", 1000), "Interval to test connectivity in milliseconds")
	flag.IntVar(&checkFailureThreshold, "threshold", getEnvInt("NAT_THRESHOLD", 5), "Number of times the check may fail before action is taken")

	flag.StringVar(&prometheusAddress, "prometheus", getEnv("NAT_PROMETHEUS", ":8080"), "Address to expose the Prometheus monitoring handler")

	flag.BoolVar(&dryRun, "dry-run", getEnvBool("NAT_DRY_RUN", true), "Prevents any side affects occuring")

	prometheus.MustRegister(checkDuration)
	prometheus.MustRegister(checkCount)
}

func main() {
	flag.Parse()

	if checkTarget == "" {
		glog.Fatalln("No health check target specified")
	}
	if subnetName == "" {
		subnetName = subnetId
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

		started := time.Now()
		checkChan := checkEndpoint(checkTarget)
		select {
		case <-time.After(checkTimeout):
			err = fmt.Errorf("Check timed out after %v", checkTimeout)
			glog.Errorf("Check timed out after %v", checkTimeout)
			checkCount.WithLabelValues(subnetName, "timeout").Inc()
		case err = <-checkChan:
		}
		checkDuration.WithLabelValues(subnetName).
			Observe(float64(time.Now().Sub(started))/float64(time.Second))

		if err == nil {
			checkCount.WithLabelValues(subnetName, "success").Inc()
			consecutiveFailures = 0
			glog.Infof("Check succeeded")
		} else {
			checkCount.WithLabelValues(subnetName, "error").Inc()
			consecutiveFailures++
			glog.Errorf("%v consecutive failures", consecutiveFailures)
		}

		if consecutiveFailures >= checkFailureThreshold {
			glog.Errorf("Consecutive failures greater than configured threshold")
			go action.Trigger(err)
			consecutiveFailures = 0
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

func getEnv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		glog.Fatalf("Failed to parse %v as integer: %v", val, err)
	}
	return intVal
}

func getEnvMs(key string, def int) time.Duration {
	return time.Millisecond * time.Duration(getEnvInt(key, def))
}

func getEnvBool(key string, def bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val == "true"
}
