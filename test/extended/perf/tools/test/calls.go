package test

import (
	"fmt"
	"time"
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FastCalls(client kubernetes.Interface, count int) []*WorkerConfig {
	worker := func() {
		_, err := client.CoreV1().ConfigMaps("openshift-kube-apiserver").Get(context.TODO(), "config", metav1.GetOptions{})
		if err != nil {
			klog.Errorf("call error=%s", err.Error())
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("fast-call-%d", i),
			WaitInterval: 1 * time.Second,
			Worker:       worker,
		})
	}

	return configs
}

func SlowCall(client kubernetes.Interface) *WorkerConfig {
	return &WorkerConfig{
		Name: "get-configmaps-all-namespaces",
		WaitInterval: 1 * time.Minute,
		Worker: func() {
			_, err := client.CoreV1().ConfigMaps(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				klog.Errorf("getAllConfigMaps error=%s", err.Error())
			}
		},
	}
}

