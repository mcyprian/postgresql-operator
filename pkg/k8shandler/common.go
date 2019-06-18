package k8shandler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func newLabels(clusterName, nodeName string) map[string]string {
	labels := map[string]string{
		"cluster-name": clusterName,
	}
	if nodeName != "" {
		labels["node-name"] = nodeName
	}
	return labels
}

func newResourceRequirements(resRequirements corev1.ResourceRequirements) corev1.ResourceRequirements {
	var requestMem *resource.Quantity
	var limitMem *resource.Quantity
	var requestCPU *resource.Quantity
	var limitCPU *resource.Quantity

	// Memory
	if resRequirements.Requests.Memory().IsZero() {
		rMem, _ := resource.ParseQuantity(defaultMemoryRequest)
		requestMem = &rMem
	} else {
		requestMem = resRequirements.Requests.Memory()
	}
	if resRequirements.Limits.Memory().IsZero() {
		lMem, _ := resource.ParseQuantity(defaultMemoryRequest)
		limitMem = &lMem
	} else {
		limitMem = resRequirements.Limits.Memory()
	}
	// CPU
	if resRequirements.Requests.Cpu().IsZero() {
		rCPU, _ := resource.ParseQuantity(defaultCPURequest)
		requestCPU = &rCPU
	} else {
		requestCPU = resRequirements.Requests.Cpu()
	}
	if resRequirements.Limits.Cpu().IsZero() {
		lCPU, _ := resource.ParseQuantity(defaultCPURequest)
		limitCPU = &lCPU
	} else {
		limitCPU = resRequirements.Limits.Cpu()
	}

	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    *limitCPU,
			"memory": *limitMem,
		},
		Requests: corev1.ResourceList{
			"cpu":    *requestCPU,
			"memory": *requestMem,
		},
	}
}
