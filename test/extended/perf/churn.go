package perf

import (
	"context"
	"flag"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"

	"github.com/openshift/origin/test/extended/perf/tools/core"
	"github.com/openshift/origin/test/extended/perf/tools/namespace"
	"github.com/openshift/origin/test/extended/perf/tools/poddensity"
	exutil "github.com/openshift/origin/test/extended/util"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type options struct {
	concurrency      int
	burst            int
	delay            time.Duration
	duration         time.Duration
	timeout          time.Duration
	longevity        time.Duration
	fixedPool        int
	podsPerNamespace int
}

var _ = g.Describe("[sig-scalability][Feature:Performance:Benchmark] Managed cluster should", func() {
	oc := exutil.NewCLI("perf-client")
	defer g.GinkgoRecover()

	var opts options
	flag.IntVar(&opts.concurrency, "churn-concurrency", 100, "number of concurrent workers")
	flag.IntVar(&opts.burst, "churn-burst", 10, "burst size")
	flag.DurationVar(&opts.delay, "churn-delay", 1*time.Second, "step delay")
	flag.DurationVar(&opts.duration, "churn-duration", 1*time.Minute, "test duration after steady state")
	flag.DurationVar(&opts.timeout, "churn-timeout", 5*time.Minute, "how long to wait for deployment/pod to be ready")
	flag.DurationVar(&opts.longevity, "churn-pod-longevity", 30*time.Second, "how long we want pod to live")
	flag.IntVar(&opts.fixedPool, "churn-namespaces", 1, "fixed namespace pool size")
	flag.IntVar(&opts.podsPerNamespace, "churn-pods-per-namespace", 1, "number of pods per namespace")
	flag.Parse()

	g.It("not experience workload disruption during namespace/pod churn", func() {
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
