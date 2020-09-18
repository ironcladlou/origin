package test

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"k8s.io/client-go/rest"
)

func DefaultReadonlyWorker(client kubernetes.Interface, namespace string, count int) []*WorkerConfig {
	worker := func() {
		if err := DefaultGetNamespace(client, namespace); err != nil {
			klog.Errorf("step error=%s", err.Error())
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("default-steps-%d", i),
			WaitInterval: 500 * time.Millisecond,
			Worker:       worker,
		})
	}

	return configs
}

func WithNewConnectionForEachWorker(config *rest.Config, namespace string, count int) []*WorkerConfig {
	workerFunc := func(client kubernetes.Interface) Worker {
		return func() {
			if err := DefaultGetNamespace(client, namespace); err != nil {
				klog.Errorf("step error=%s", err.Error())
			}
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		name := fmt.Sprintf("default-steps-%d", i)
		client, err := WithNewUserAgent(config, name)
		if err != nil {
			panic(err)
		}

		configs = append(configs, &WorkerConfig{
			Name:         name,
			WaitInterval: 1 * time.Millisecond,
			Worker:       workerFunc(client),
		})
	}

	return configs
}


func DefaultGetNamespace(client kubernetes.Interface, namespace string) error {
	_, err := client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return nil
}


func WithNewUserAgent(config *rest.Config, userAgent string) (client kubernetes.Interface, err error) {
	copy := rest.CopyConfig(config)
	copy = rest.AddUserAgent(copy, userAgent)

	client, err = kubernetes.NewForConfig(copy)
	return
}
