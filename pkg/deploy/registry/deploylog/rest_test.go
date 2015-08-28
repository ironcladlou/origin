package deploylog

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client"
	ktestclient "k8s.io/kubernetes/pkg/client/testclient"
	genericrest "k8s.io/kubernetes/pkg/registry/generic/rest"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/openshift/origin/pkg/client/testclient"
	"github.com/openshift/origin/pkg/deploy/api"
	deploytest "github.com/openshift/origin/pkg/deploy/api/test"
	deployutil "github.com/openshift/origin/pkg/deploy/util"
)

func makeDeployment(version int) kapi.ReplicationController {
	deployment, _ := deployutil.MakeDeployment(deploytest.OkDeploymentConfig(version), kapi.Codec)
	return *deployment
}

func makeDeploymentList(versions int) *kapi.ReplicationControllerList {
	list := &kapi.ReplicationControllerList{}
	for v := 1; v <= versions; v++ {
		list.Items = append(list.Items, makeDeployment(v))
	}
	return list
}

// Mock pod resource getter
type deployerPodGetter struct{}

func (p *deployerPodGetter) Get(ctx kapi.Context, name string) (runtime.Object, error) {
	return &kapi.Pod{
		ObjectMeta: kapi.ObjectMeta{
			Name:      name,
			Namespace: kapi.NamespaceDefault,
		},
		Spec: kapi.PodSpec{
			Containers: []kapi.Container{
				{
					Name: name + "-container",
				},
			},
			NodeName: name + "-host",
		},
	}, nil
}

// mockREST mocks a DeploymentLog REST
func mockREST(version, desired int, endStatus api.DeploymentStatus) *REST {
	// Fake deploymentConfig
	fakeDn := testclient.NewSimpleFake(deploytest.OkDeploymentConfig(version))
	// Fake deployments
	fakeDeployments := makeDeploymentList(version)
	fakeRn := ktestclient.NewSimpleFake(fakeDeployments)
	// Fake watcher for deployments
	fakeRn.Watch = watch.NewFake()
	// Everything is fake
	connectionInfo := &kclient.HTTPKubeletClient{Config: &kclient.KubeletConfig{EnableHttps: true, Port: 12345}, Client: &http.Client{}}

	go func(obj *kapi.ReplicationController) {
		// Add deployment in the fake watcher - status New
		fakeRn.Watch.(*watch.FakeWatcher).Add(obj)
		time.Sleep(100 * time.Millisecond)

		// Modify deployment in the fake watcher - status Pending
		obj.Annotations[api.DeploymentStatusAnnotation] = string(api.DeploymentStatusPending)
		fakeRn.Watch.(*watch.FakeWatcher).Modify(obj)
		time.Sleep(100 * time.Millisecond)

		// Modify deployment in the fake watcher - status depends on endStatus
		// Running, Complete, or Failed, all should be tested
		obj.Annotations[api.DeploymentStatusAnnotation] = string(endStatus)
		fakeRn.Watch.(*watch.FakeWatcher).Modify(obj)
	}(&fakeDeployments.Items[desired-1])

	return &REST{
		DeploymentConfigsNamespacer:      fakeDn,
		ReplicationControllersNamespacer: fakeRn,
		PodGetter:                        &deployerPodGetter{},
		ConnectionInfo:                   connectionInfo,
		Timeout:                          defaultTimeout,
	}
}

func TestRESTGet(t *testing.T) {
	ctx := kapi.NewDefaultContext()

	tests := []struct {
		testName    string
		rest        *REST
		name        string
		opts        runtime.Object
		expected    runtime.Object
		expectedErr error
	}{
		{
			testName: "running deployment",
			rest:     mockREST(1, 1, api.DeploymentStatusRunning),
			name:     "config",
			opts:     &api.DeploymentLogOptions{Follow: true, Version: 1},
			expected: &genericrest.LocationStreamer{
				Location: &url.URL{
					Scheme:   "https",
					Host:     "config-1-deploy-host:12345",
					Path:     "/containerLogs/default/config-1-deploy/config-1-deploy-container",
					RawQuery: "follow=true",
				},
				Transport:   nil,
				ContentType: "text/plain",
				Flush:       true,
			},
			expectedErr: nil,
		},
		// TODO: Uncomment once the testclient supports GETing objects that are added by NewSimpleFake
		/*{
			testName: "complete deployment",
			rest:     mockREST(5, 5, api.DeploymentStatusComplete),
			name:     "config",
			opts:     &api.DeploymentLogOptions{Follow: true, Version: 5},
			expected: &genericrest.LocationStreamer{
				Location: &url.URL{
					Scheme:   "https",
					Host:     "config-5-deploy-host:12345",
					Path:     "/containerLogs/default/config-5-deploy/config-5-deploy-container",
					RawQuery: "follow=true",
				},
				Transport:   nil,
				ContentType: "text/plain",
				Flush:       true,
			},
			expectedErr: nil,
		},
		{
			testName: "previous failed deployment",
			rest:     mockREST(3, 2, api.DeploymentStatusFailed),
			name:     "config",
			opts:     &api.DeploymentLogOptions{Follow: false, Version: 2},
			expected: &genericrest.LocationStreamer{
				Location: &url.URL{
					Scheme: "https",
					Host:   "config-2-deploy-host:12345",
					Path:   "/containerLogs/default/config-2-deploy/config-2-deploy-container",
				},
				Transport:   nil,
				ContentType: "text/plain",
				Flush:       false,
			},
			expectedErr: nil,
		},*/
	}

	for _, test := range tests {
		got, err := test.rest.Get(ctx, test.name, test.opts)
		if err != test.expectedErr {
			t.Errorf("%s: error mismatch: expected %v, got %v", test.testName, test.expectedErr, err)
			continue
		}
		if !reflect.DeepEqual(got, test.expected) {
			t.Errorf("%s: location streamer mismatch: expected\n%#v\ngot\n%#v\n", test.testName, test.expected, got)
			if testing.Verbose() {
				e := test.expected.(*genericrest.LocationStreamer)
				a := got.(*genericrest.LocationStreamer)
				t.Errorf("%s: expected url:\n%v\ngot:\n%v\n", test.testName, e.Location, a.Location)
			}
		}
	}
}
