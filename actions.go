package main

import (
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	actionTriggerDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "natcheck_action_duration_milliseconds",
		Help: "The time taken to trigger each action",
		Buckets: prometheus.LinearBuckets(0, 1000, 10),
	},
		[]string{"subnet", "action"},
	)
	actionTriggerResults = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "natcheck_action_total",
		Help: "The count of the results of triggering each action",
	},
		[]string{"subnet", "action", "result"},
	)
)

func init() {
	prometheus.MustRegister(actionTriggerDuration)
	prometheus.MustRegister(actionTriggerResults)
}

type Action interface {
	Trigger(error) error
}

type FanoutAction struct {
	actions map[string]Action
}

func newFanoutAction() *FanoutAction {
	return &FanoutAction{
		actions: make(map[string]Action),
	}
}

func (fa *FanoutAction) AddAction(name string, action Action) {
	if action != nil {
		fa.actions[name] = action
	}
}

func (fa *FanoutAction) Trigger(upstreamErr error) error {
	glog.Infof("Async fanning out %v actions", len(fa.actions))

	for name, act := range fa.actions {
		go func(name string, act Action) {
			started := time.Now()
			err := act.Trigger(upstreamErr)
			actionTriggerDuration.
				WithLabelValues(subnetName, name).
				Observe(float64(time.Now().Sub(started) / time.Millisecond))

			if err != nil {
				glog.Errorf("Action %v failed: %v", name, err)
				actionTriggerResults.WithLabelValues(subnetName, name, "error").Inc()
			} else {
				glog.Infof("Action %v succeeded", name)
				actionTriggerResults.WithLabelValues(subnetName, name, "success").Inc()
			}
		}(name, act)
	}
	return nil
}

type statelessAction struct {
	f func(error) error
}

func (s statelessAction) Trigger(err error) error {
	return s.f(err)
}

func makeAction(f func(error) error) Action {
	return statelessAction{f}
}
