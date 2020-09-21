package perf

import (
	"context"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	"github.com/openshift/origin/test/extended/perf/tools/core"
	"github.com/openshift/origin/test/extended/perf/tools/namespace"
	"github.com/openshift/origin/test/extended/perf/tools/poddensity"
	exutil "github.com/openshift/origin/test/extended/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type options struct {
	// number of concurrent workers
	concurrency int
	// burst size
	burst int
	// step delay
	delay time.Duration
	// test duration after steady state
	duration time.Duration
	// how long to wait for the deployment/pod to be ready
	timeout time.Duration
	// how long pods should live
	longevity time.Duration
	// fixed namespace pool size
	fixedPool int
	// number of pods per namespace
	podsPerNamespace int
}

var ipiChurnDefaults = options{
	concurrency:      30,
	burst:            10,
	delay:            1 * time.Second,
	duration:         10 * time.Minute,
	timeout:          5 * time.Minute,
	longevity:        60 * time.Second,
	fixedPool:        30,
	podsPerNamespace: 0,
}

var _ = g.Describe("[sig-scalability][Feature:Performance:Benchmark] Managed cluster should", func() {
	oc := exutil.NewCLI("perf-client")
	defer g.GinkgoRecover()

	g.It("not experience workload disruption during namespace/pod churn", func() {
		opts := ipiChurnDefaults

		client := oc.AdminKubeClient()

		config := oc.AdminConfig()
		config.QPS = 15000
		config.Burst = 20000

		// TODO: implement cancellation
		shutdown, cancel := context.WithCancel(context.TODO())
		defer cancel()
		tc, testCancel := core.NewTestContext(shutdown, opts.duration)
		defer testCancel()

		var pool namespace.Pool
		if opts.podsPerNamespace > 0 {
			g.By("using a churning namespace pool")
			p, err := namespace.NewPoolWithChurn(config, opts.podsPerNamespace)
			o.Expect(err).NotTo(o.HaveOccurred())

			pool = p
		} else {
			g.By("using a fixed namespace pool")
			p, err := namespace.NewFixedPool(client, opts.fixedPool)
			o.Expect(err).NotTo(o.HaveOccurred())

			pool = p
		}
		defer func() {
			err := pool.Dispose()
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to clean up namespace pool")
		}()

		// setup a dummy worker
		worker := poddensity.NewWorker(client, pool.GetNamespace, opts.timeout, opts.longevity)

		// run this worker in parallel
		runner := core.NewRunnerWithDelay(1 * time.Millisecond)

		actions := runner.ToActions(tc, opts.concurrency, worker, "pod-density")
		generator := core.NewSteppedLoadGenerator(opts.delay, opts.burst)

		g.By("simulating a workload")
		go generator.Generate(actions)

		<-tc.TestCancel.Done()

		e2e.Logf("test duration elapsed, waiting for worker(s) to be done")
		tc.WaitGroup.Wait()
		e2e.Logf("all worker(s) are done")

		// TODO: just a basic sanity check to measure success
		g.By("ensuring all nodes remain ready")
		nodes, err := oc.AdminKubeClient().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		notReadyNodes := sets.NewString()
		for _, node := range nodes.Items {
			for _, c := range node.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionFalse {
					notReadyNodes.Insert(node.Name)
				}
			}
		}
		o.Expect(notReadyNodes).To(o.BeEmpty())
	})
})
