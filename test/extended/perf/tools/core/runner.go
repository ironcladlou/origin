package core

import (
	"fmt"
	"k8s.io/klog"
	"time"
)

func NewRunnerWithDelay(delay time.Duration) Runner {
	return func(wc *WorkerContext, worker Worker) {
		defer func() {
			if wc.WaitGroup!= nil {
				wc.WaitGroup.Done()
			}
			klog.V(5).Infof("worker=%s - worker loop done", wc.Name)
		}()

		klog.V(5).Infof("worker=%s - worker loop started", wc.Name)

		for {
			select {
			case <-wc.Shutdown.Done():
				return
			default:
				<-time.After(delay)
				worker.Work(wc)
			}
		}
	}
}

func (r Runner)  ToActions(tc *TestContext, concurrency int, worker Worker, prefix string) []Action {
	doers := make([]Action, 0)

	if concurrency <= 0 {
		return doers
	}

	if tc.WaitGroup != nil {
		tc.WaitGroup.Add(concurrency)
	}

	for i := 1; i <= concurrency; i++ {
		wc := WorkerContext{
			Name: fmt.Sprintf("%s-%d", prefix, i),
			WaitGroup: tc.WaitGroup,
			Shutdown: tc.TestCancel,
		}

		doers = append(doers, func() {
			r.Run(&wc, worker)
		})
	}

	return doers
}
