package test

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/klog"
)

type EventHandler func(event *corev1.Event)

type Watcher interface {
	Start()
}

func (h EventHandler) Handle(event *corev1.Event) {
	h(event)
}

func NewEventWatcher(factory informers.SharedInformerFactory, handler EventHandler) {
	informer := factory.Core().V1().Events().Informer()
	informer.AddEventHandler(prepareHandler(handler))
}

func prepareHandler(handler EventHandler) cache.ResourceEventHandler {
	handle := func(obj interface{}) {
		event, ok := obj.(*corev1.Event)
		if !ok {
			klog.Errorf("[EventWatcher] EventHandler: object is not of Event type, type=%T", obj)
			return
		}

		handler.Handle(event)
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handle(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			handle(new)
		},
	}
}
