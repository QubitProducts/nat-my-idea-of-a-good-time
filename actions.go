package main

import (
	"github.com/golang/glog"
)

type Action interface {
	Trigger(error)
}

type FanoutAction struct {
	actions []Action
}

func newFanoutAction() *FanoutAction {
	return &FanoutAction{}
}

func (fa *FanoutAction) AddAction(action Action) {
	if action != nil {
		fa.actions = append(fa.actions, action)
	}
}

func (fa *FanoutAction) Trigger(err error) {
	glog.Infof("Async fanning out %v actions", len(fa.actions))

	for _, a := range fa.actions {
		go a.Trigger(err)
	}
}

type statelessAction struct {
	f func(error)
}

func (s statelessAction) Trigger(err error) {
	s.f(err)
}

func makeAction(f func(error)) Action {
	return statelessAction{f}
}
