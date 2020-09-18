package test

//import (
//	"fmt"
//	"k8s.io/apimachinery/pkg/util/intstr"
//	"k8s.io/apimachinery/pkg/util/rand"
//	"k8s.io/apimachinery/pkg/api/resource"
//	"k8s.io/apimachinery/pkg/util/wait"
//
//	"time"
//	"context"
//
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	corev1 "k8s.io/api/core/v1"
//	appsv1 "k8s.io/api/apps/v1"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/klog"
//)
//
//func DefaultDeploymentWorker(client kubernetes.Interface, namespace string, count int) []*WorkerConfig {
//	worker := func() {
//		if err := DefaultSteps(namespace, client); err != nil {
//			klog.Errorf("step error=%s", err.Error())
//		}
//	}
//
//	configs := make([]*WorkerConfig, 0)
//	for i := 1; i <=count; i++ {
//		configs = append(configs, &WorkerConfig{
//			Name:         fmt.Sprintf("default-steps-%d", i),
//			WaitInterval: 1 * time.Millisecond,
//			Worker:       worker,
//		})
//	}
//
//	return configs
//}
//
//
//type deployment struct {
//
//}
//
//func (d *deployment) create(client kubernetes.Interface) error {
//	prefix := "test-"
//	ns, err := client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
//		ObjectMeta: metav1.ObjectMeta{
//			GenerateName: prefix,
//			Labels: map[string]string{
//				"clusterloader": "true",
//			},
//		},
//	}, metav1.CreateOptions{})
//	if err != nil {
//		return err
//	}
//
//	namespace := ns.GetName()
//	sa, err := client.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), &corev1.ServiceAccount{
//		ObjectMeta: metav1.ObjectMeta{
//			GenerateName: prefix,
//			Labels: map[string]string{
//				"clusterloader": "true",
//			},
//		},
//	}, metav1.CreateOptions{})
//	if err != nil {
//		return err
//	}
//
//	secret, err := client.CoreV1().Secrets(namespace).Create(context.TODO(), &corev1.Secret{
//		ObjectMeta: metav1.ObjectMeta{
//			GenerateName: prefix,
//			Labels: map[string]string{
//				"clusterloader": "true",
//			},
//		},
//		Type: corev1.SecretTypeOpaque,
//		Data: map[string][]byte{
//			"key1": []byte("aGVsbG8gd29ybGQgMQo="),
//			"key2": []byte("aGVsbG8gd29ybGQgMgo="),
//		},
//	}, metav1.CreateOptions{})
//	if err != nil {
//		return err
//	}
//
//	cm, err := client.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			GenerateName: prefix,
//			Labels: map[string]string{
//				"clusterloader": "true",
//			},
//		},
//		Data: map[string]string{
//			"key1": "foo",
//			"key2": "bar",
//		},
//	}, metav1.CreateOptions{})
//	if err != nil {
//		return err
//	}
//
//	deployment := new(namespace, prefix, sa.GetName(), cm.GetName(), secret.GetName())
//	deployment, err = client.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
//	return err
//}
//
//func (d *deployment) waitAndDestroy() {
//	err = wait.Poll(100 * time.Millisecond, WaitPollTimeout, func() (done bool, pollErr error) {
//		deployment, pollErr := client.AppsV1().Deployments(namespace).Get(context.TODO(), deployment.GetName(), metav1.GetOptions{})
//		if pollErr != nil {
//			return
//		}
//
//		if cluster == nil || !f(cluster) {
//			return
//		}
//
//		done = true
//		return
//	})
//}
//
//
//
//
//
//	if _, err := client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), sa.GetName(), metav1.GetOptions{}); err != nil {
//		return err
//	}
//
//	if _, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secret.GetName(), metav1.GetOptions{}); err != nil {
//		return err
//	}
//
//	if _, err := client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), cm.GetName(), metav1.GetOptions{}); err != nil {
//		return err
//	}
//
//	if sa != nil {
//		if err := client.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), sa.GetName(), metav1.DeleteOptions{}); err != nil {
//			return err
//		}
//	}
//
//	if secret != nil {
//		if err := client.CoreV1().Secrets(namespace).Delete(context.TODO(), secret.GetName(), metav1.DeleteOptions{}); err != nil {
//			return err
//		}
//	}
//
//	if cm != nil {
//		if err := client.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), cm.GetName(), metav1.DeleteOptions{}); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//
//func new(namespace, prefix, sa, cm, secret string) *appsv1.Deployment {
//	var replicas int32 = 1
//	name := fmt.Sprintf("%s-%s", prefix, rand.String(10))
//
//	return &appsv1.Deployment{
//		TypeMeta: metav1.TypeMeta{
//			Kind:       "Deployment",
//			APIVersion: "apps/v1",
//		},
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: namespace,
//			Name: name,
//			Labels: map[string]string{
//				"clusterloader": "true",
//			},
//		},
//		Spec: appsv1.DeploymentSpec{
//			Replicas: &replicas,
//			Selector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"clusterloader": name,
//				},
//			},
//			Template: corev1.PodTemplateSpec{
//				ObjectMeta: metav1.ObjectMeta{
//					Name: namespace,
//					Labels: map[string]string{
//						"clusterloader": name,
//					},
//				},
//				Spec: corev1.PodSpec{
//					ServiceAccountName: sa,
//					Containers: []corev1.Container{
//						{
//							Name:            "clusterloader",
//							Image:           "k8s.gcr.io/pause:3.1",
//							ImagePullPolicy: corev1.PullIfNotPresent,
//							Resources: corev1.ResourceRequirements{
//								Requests: corev1.ResourceList{
//									corev1.ResourceCPU: resource.MustParse("10m"),
//									corev1.ResourceMemory: resource.MustParse("10m"),
//								},
//							},
//							VolumeMounts: []corev1.VolumeMount{
//								{
//									Name:      "configmap",
//									MountPath: "/var/configmap",
//								},
//								{
//									Name:      "secret",
//									MountPath: "/var/secret",
//								},
//							},
//						},
//					},
//					Volumes: []corev1.Volume{
//						{
//							Name: "secret",
//							VolumeSource: corev1.VolumeSource{
//								Secret: &corev1.SecretVolumeSource{
//									SecretName: secret,
//									DefaultMode: func() *int32 {
//										v := int32(420)
//										return &v
//									}(),
//								},
//							},
//						},
//						{
//							Name: "configmap",
//							VolumeSource: corev1.VolumeSource{
//								ConfigMap: &corev1.ConfigMapVolumeSource{
//									LocalObjectReference: corev1.LocalObjectReference{
//										Name: cm,
//									},
//								},
//							},
//						},
//					},
//					Tolerations: []corev1.Toleration{
//						{
//							Key: "node.kubernetes.io/not-ready",
//							Operator: corev1.TolerationOpExists,
//							Effect: corev1.TaintEffectNoExecute,
//							TolerationSeconds: func() *int64 {
//								v := int64(900)
//								return &v
//							}(),
//						},
//						{
//							Key: "node.kubernetes.io/unreachable",
//							Operator: corev1.TolerationOpExists,
//							Effect: corev1.TaintEffectNoExecute,
//							TolerationSeconds: func() *int64 {
//								v := int64(900)
//								return &v
//							}(),
//						},
//					},
//				},
//			},
//		},
//	}
//}
