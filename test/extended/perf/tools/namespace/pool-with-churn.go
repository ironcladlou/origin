package namespace

import (
	"context"
	"fmt"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/apimachinery/pkg/util/rand"

	projectv1 "github.com/openshift/api/project/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned"
)

func NewPoolWithChurn(config *rest.Config, maxPodsPerNamespace int) (*PoolWithChurn, error) {
	client, err := projectv1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	p := &PoolWithChurn{
		client:          client,
		maxPerNamespace: maxPodsPerNamespace,
		pool: map[string]*projectWithUsage{},
	}
	return p, nil
}

type projectWithUsage struct {
	project *projectv1.Project

	lock      sync.Mutex
	remaining int
	done      int
}

func (p *projectWithUsage) Done() int {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.done--
	return p.done
}

func (p *projectWithUsage) Use() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.remaining == 0 {
		return false
	}

	p.remaining--
	return true
}

type PoolWithChurn struct {
	client          projectv1client.Interface
	maxPerNamespace int

	lock sync.Mutex
	pool map[string]*projectWithUsage
}

func (p *PoolWithChurn) remove(namespace string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.pool) > 0 {
		delete(p.pool, namespace)
	}
}

func (p *PoolWithChurn) GetNamespace() (namespace string, done Done, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var pc *projectWithUsage
	for _, v := range p.pool {
		if v.Use() {
			pc = v
			break
		}
	}

	if pc == nil {
		project, createErr := createProject("test", p.client)
		if createErr != nil {
			err = createErr
			return
		}

		pc = &projectWithUsage{
			project:   project,
			remaining: p.maxPerNamespace,
			done:      p.maxPerNamespace,
		}
		pc.Use()
		p.pool[project.GetName()] = pc
	}

	namespace = pc.project.GetName()
	done = func() {
		// TODO: what's the purpose of pc.Done() given this is the only call site and the
		// user already indicated doneness by calling this function?
		if pc.Done() == 0 {
			// TODO: If the pool serves up another namespace prior to delete completion/success
			// is it possible for invariants to be violated re: max namespaces the pool should have?
			p.remove(namespace)

			if err := p.client.ProjectV1().Projects().Delete(context.TODO(), namespace, metav1.DeleteOptions{}); err != nil {
				if k8serrors.IsNotFound(err) {
					return
				}

				klog.Errorf("[PoolWithChurn] failed to delete project name=%s - %s", namespace, err.Error())
			}
		}
	}
	return
}

func (p *PoolWithChurn) Dispose() error {
	if len(p.pool) > 0 {
		klog.Infof("[PoolWithChurn] deleting remaining projects size=%d", len(p.pool))
	}
	for name, _ := range p.pool {
		delete(p.pool, name)
		if err := p.client.ProjectV1().Projects().Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
			if k8serrors.IsNotFound(err) {
				continue
			}

			klog.Errorf("[PoolWithChurn] failed to delete project name=%s - %s", name, err.Error())
		}
	}
	klog.Infof("[PoolWithChurn] remaining projects size=%d", len(p.pool))
	return nil
}

func createProject(prefix string, client projectv1client.Interface) (*projectv1.Project, error) {
	// now create a project
	name := fmt.Sprintf("%s-%s", prefix, rand.String(10))
	project := &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
	}

	return client.ProjectV1().Projects().Create(context.TODO(), project, metav1.CreateOptions{})
}
