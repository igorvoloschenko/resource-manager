package kube

import (
	"context"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Clientset struct {
	*kubernetes.Clientset
}

var clientset *Clientset

func NewClientset() (*Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			kubeconfig = ""
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	return &Clientset{clientset}, err
}

func Init() (err error) {
	clientset, err = NewClientset()
	return
}

func GetNamespace(nsName string) (*corev1.Namespace, error) {
	return clientset.CoreV1().Namespaces().Get(
		context.Background(),
		nsName,
		metav1.GetOptions{},
	)
}

func GetNamespaces() (*corev1.NamespaceList, error) {
	return clientset.CoreV1().Namespaces().List(
		context.Background(),
		metav1.ListOptions{},
	)
}

func CreateQuota(rq *corev1.ResourceQuota) (*corev1.ResourceQuota, error) {
	return clientset.CoreV1().ResourceQuotas(rq.Namespace).Create(
		context.Background(),
		rq,
		metav1.CreateOptions{},
	)
}

func UpdateQuota(rq *corev1.ResourceQuota) (*corev1.ResourceQuota, error) {
	return clientset.CoreV1().ResourceQuotas(rq.Namespace).Update(
		context.Background(),
		rq,
		metav1.UpdateOptions{},
	)
}

func DeleteQuota(rq *corev1.ResourceQuota) error {
	return clientset.CoreV1().ResourceQuotas(rq.Namespace).Delete(
		context.Background(),
		rq.GetName(),
		metav1.DeleteOptions{},
	)
}

func GetQuota(name, ns string) (*corev1.ResourceQuota, error) {
	return clientset.CoreV1().ResourceQuotas(ns).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

func CreateLimitRanges(lr *corev1.LimitRange) (*corev1.LimitRange, error) {
	return clientset.CoreV1().LimitRanges(lr.Namespace).Create(
		context.Background(),
		lr,
		metav1.CreateOptions{},
	)
}

func UpdateLimitRanges(lr *corev1.LimitRange) (*corev1.LimitRange, error) {
	return clientset.CoreV1().LimitRanges(lr.Namespace).Update(
		context.Background(),
		lr,
		metav1.UpdateOptions{},
	)
}

func DeleteLimitRanges(lr *corev1.LimitRange) error {
	return clientset.CoreV1().LimitRanges(lr.Namespace).Delete(
		context.Background(),
		lr.GetName(),
		metav1.DeleteOptions{},
	)
}
