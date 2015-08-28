package client

import (
	kapi "k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/conversion/queryparams"

	"github.com/openshift/origin/pkg/deploy/api"
)

// DeploymentLogsNamespacer has methods to work with DeploymentLogs resources in a namespace
type DeploymentLogsNamespacer interface {
	DeploymentLogs(namespace string) DeploymentLogInterface
}

// DeploymentLogInterface exposes methods on DeploymentLogs resources.
type DeploymentLogInterface interface {
	Get(name string, opts api.DeploymentLogOptions) (*kclient.Request, error)
}

// deploymentLogs implements DeploymentLogsNamespacer interface
type deploymentLogs struct {
	r  *Client
	ns string
}

// newDeploymentLogs returns a deploymentLogs
func newDeploymentLogs(c *Client, namespace string) *deploymentLogs {
	return &deploymentLogs{
		r:  c,
		ns: namespace,
	}
}

// Get gets the deploymentlogs and return a deploymentLog request
func (c *deploymentLogs) Get(name string, opts api.DeploymentLogOptions) (*kclient.Request, error) {
	req := c.r.Get().Namespace(c.ns).Resource("deploymentConfigs").Name(name).SubResource("log")

	versioned, err := kapi.Scheme.ConvertToVersion(&opts, c.r.APIVersion())
	if err != nil {
		return nil, err
	}
	params, err := queryparams.Convert(versioned)
	if err != nil {
		return nil, err
	}
	for k, v := range params {
		for _, vv := range v {
			req.Param(k, vv)
		}
	}

	return req, nil
}
