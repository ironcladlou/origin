package test

import (
	"fmt"
	"strconv"
	"time"
	"context"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func HealthCheckWorker(client clientset.Interface, count int) []*WorkerConfig {
	healthz := func() {
		if err := RunHealthzProbe(client); err != nil {
			klog.Errorf("/healthz error=%s", err.Error())
		}
	}

	readyz := func() {
		if err := RunReadyzProbe(client); err != nil {
			klog.Errorf("/readyz error=%s", err.Error())
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("healthz-steps-%d", i),
			WaitInterval: 1 * time.Millisecond,
			Worker:       healthz,
		})
	}

	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("readyz-steps-%d", i),
			WaitInterval: 1 * time.Millisecond,
			Worker:       readyz,
		})
	}

	return configs
}

func RunHealthzProbe(client clientset.Interface) error {
	healthStatus := 0
	result := client.Discovery().RESTClient().Get().AbsPath("/healthz").Do(context.TODO()).StatusCode(&healthStatus)

	if result.Error() != nil {
		Increment("<error>", "healthz", "localhost");
		return result.Error()
	}

	Increment(strconv.FormatInt(int64(healthStatus), 10), "healthz", "localhost");
	return nil
}

func RunReadyzProbe(client clientset.Interface) error {
	healthStatus := 0
	result := client.Discovery().RESTClient().Get().AbsPath("/readyz").Do(context.TODO()).StatusCode(&healthStatus)

	if result.Error() != nil {
		Increment("<error>", "readyz", "localhost");
		return result.Error()
	}

	Increment(strconv.FormatInt(int64(healthStatus), 10), "readyz", "localhost");
	return nil
}

