package namespace

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func NewFixedPool(client kubernetes.Interface, size int) (*FixedPool, error) {
	pool := []*corev1.Namespace{}
	prefix := "test-"

	for i := 1; i <= size; i++ {
		ns, err := client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: prefix,
				Labels: map[string]string{
					"clusterloader": "true",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}

		pool = append(pool, ns)
	}

	return &FixedPool{
		client: client,
		pool:   pool,
	}, nil
}

type FixedPool struct {
	pool   []*corev1.Namespace
	client kubernetes.Interface
}

func (p *FixedPool) GetNamespace() (namespace string, done Done, err error) {
	// Fixed pool of namespace, the caller does not need to notify us
	// when it is done using the namespace.
	done = func() {}

	if len(p.pool) == 0 {
		return
	}

	i := rand.Intn(len(p.pool))
	namespace = p.pool[i].GetName()
	return
}

func (p *FixedPool) Dispose() error {
	klog.Infof("[FixedPool] deleting namespaces size=%d", len(p.pool))
	for i := range p.pool {
		ns := p.pool[i]
		if err := p.client.CoreV1().Namespaces().Delete(context.TODO(), ns.GetName(), metav1.DeleteOptions{}); err != nil {
			if k8serrors.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("deleted=%d target=%d error deleting namespace: %s", i+1, len(p.pool)-i, err.Error())
		}
	}

	return nil
}
