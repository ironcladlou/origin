package test

import (
	"fmt"
	"time"
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func DefaultStepsWorker(client kubernetes.Interface, namespace string, count int) []*WorkerConfig {
	worker := func() {
		if err := DefaultSteps(namespace, client); err != nil {
			klog.Errorf("step error=%s", err.Error())
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("default-steps-%d", i),
			WaitInterval: 1 * time.Millisecond,
			Worker:       worker,
		})
	}

	return configs
}

func DefaultSteps(namespace string, client kubernetes.Interface) error {
	prefix := "test-"
	sa, err := client.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
	}, metav1.CreateOptions{} )
	if err != nil {
		return err
	}

	secret, err := client.CoreV1().Secrets(namespace).Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key1": []byte("aGVsbG8gd29ybGQgMQo="),
			"key2": []byte("aGVsbG8gd29ybGQgMgo="),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	cm, err := client.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
		Data: map[string]string{
			"key1": "foo",
			"key2": "bar",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if _, err := client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), sa.GetName(), metav1.GetOptions{}); err != nil {
		return err
	}

	if _, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secret.GetName(), metav1.GetOptions{}); err != nil {
		return err
	}

	if _, err := client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), cm.GetName(), metav1.GetOptions{}); err != nil {
		return err
	}

	if sa != nil {
		if err := client.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), sa.GetName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	if secret != nil {
		if err := client.CoreV1().Secrets(namespace).Delete(context.TODO(), secret.GetName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	if cm != nil {
		if err := client.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), cm.GetName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}
