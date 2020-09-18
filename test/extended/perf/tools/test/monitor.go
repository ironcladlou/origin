package test

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func MonitorWorker(client kubernetes.Interface) []*WorkerConfig {
	worker := func() {
		if err := getNamespace(client, "kube-system"); err != nil {
			klog.Errorf("monitor test error=%s", err.Error())
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=1; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("monitor-test-%d", i),
			WaitInterval: 500 * time.Millisecond,
			Worker:       worker,
		})
	}

	return configs
}

func getNamespace(client kubernetes.Interface, namespace string) error {
	_, err := client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return nil
}
