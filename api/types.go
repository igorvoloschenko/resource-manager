package api

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BodyResourceQuota struct {
	MetaData metav1.ObjectMeta        `json:"metadata"`
	Spec     corev1.ResourceQuotaSpec `json:"spec"`
}

type BodyLimitRange struct {
	MetaData metav1.ObjectMeta     `json:"metadata"`
	Spec     corev1.LimitRangeSpec `json:"spec"`
}
