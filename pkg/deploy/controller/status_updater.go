package controller

import (
	"fmt"

	"k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/client/record"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"

	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deployutil "github.com/openshift/origin/pkg/deploy/util"
)

type DeploymentStatusUpdater interface {
	UpdateStatus(deployment *api.ReplicationController, desiredStatus deployapi.DeploymentStatus) (*api.ReplicationController, error)
}

type DefaultStatusUpdater struct {
	Recorder     record.EventRecorder
	KubeClient   kclient.Interface
	DecodeConfig func(deployment *api.ReplicationController) (*deployapi.DeploymentConfig, error)
}

func (u *DefaultStatusUpdater) UpdateStatus(deployment *api.ReplicationController, desiredStatus deployapi.DeploymentStatus) (*api.ReplicationController, error) {
	originalStatus := deployutil.DeploymentStatusFor(deployment)
	tries := 0
	retryErr := kclient.RetryOnConflict(kclient.DefaultRetry, func() error {
		currentStatus := deployutil.DeploymentStatusFor(deployment)
		if !deployutil.CanTransitionPhase(currentStatus, desiredStatus) {
			return nil
		}
		tries = tries + 1
		fmt.Printf("@@@@@@@@@@@@ Updating status for %s (attempt %d)\n", deployment.Name, tries)
		deployment.Annotations[deployapi.DeploymentStatusAnnotation] = string(desiredStatus)
		updated, updateErr := u.KubeClient.ReplicationControllers(deployment.Namespace).Update(deployment)
		if updateErr == nil {
			deployment = updated
			return nil
		}
		if !kerrors.IsConflict(updateErr) {
			return updateErr
		}
		u.Recorder.Eventf(deployment, api.EventTypeWarning, "FailedUpdate", "Cannot update status from %s to %s due to an update conflict", originalStatus, desiredStatus)
		// On conflict, get the new deployment state to retry.
		existing, getErr := u.KubeClient.ReplicationControllers(deployment.Namespace).Get(deployment.Name)
		// If we can't get the latest deployment, stop retrying.
		if getErr != nil {
			return getErr
		}
		// Update local state for conflict retry.
		deployment = existing
		return updateErr
	})
	if retryErr != nil {
		if config, decodeErr := u.DecodeConfig(deployment); decodeErr == nil {
			u.Recorder.Eventf(config, api.EventTypeWarning, "FailedUpdate", "Cannot update deployment %s status to %s: %v", deployutil.LabelForDeployment(deployment), desiredStatus, retryErr)
		} else {
			u.Recorder.Eventf(deployment, api.EventTypeWarning, "FailedUpdate", "Cannot update deployment %s status to %s: %v", deployutil.LabelForDeployment(deployment), desiredStatus, retryErr)
		}
		return nil, fmt.Errorf("couldn't update Deployment %s to status %s: %v", deployutil.LabelForDeployment(deployment), desiredStatus, retryErr)
	}
	if newStatus := deployutil.DeploymentStatusFor(deployment); originalStatus != newStatus {
		u.Recorder.Eventf(deployment, api.EventTypeNormal, "DeploymentStatusUpdated", "Status updated from %s to %s", originalStatus, newStatus)
	}
	return deployment, nil
}
