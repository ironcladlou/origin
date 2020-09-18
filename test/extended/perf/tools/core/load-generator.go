package core

import (
	"context"
	"time"
	"golang.org/x/time/rate"

	"k8s.io/klog"
)

func NewSteppedLoadGenerator(delay time.Duration, burst int) LoadGenerator {
	return func(actions []Action) {
		for i := range actions {
			action := actions[i]
			go action()

			if (i+1) % burst == 0 {
				klog.V(4).Infof("done=%d target=%d, waiting for %s", i+1, len(actions), delay)
				<-time.After(delay)
			}
		}
	}
}

func NewRateLimitedLoadGenerator(qps, burst int) LoadGenerator {
	limiter := rate.NewLimiter(rate.Limit(qps), burst)
	return func(actions []Action) {
		for i := range actions {
			action := actions[i]
			limiter.Wait(context.TODO())
			go action()
		}
	}
}

