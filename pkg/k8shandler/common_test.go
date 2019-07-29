package k8shandler

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNewLabels(t *testing.T) {
	table := []struct {
		clusterName string
		nodeName    string
		primary     bool
		expected    map[string]string
	}{
		{
			"example-cluster",
			"example-node",
			false,
			map[string]string{
				"cluster-name": "example-cluster",
				"node-name":    "example-node",
			}},
		{
			"example-cluster",
			"example-node",
			true,
			map[string]string{
				"cluster-name": "example-cluster",
				"node-name":    "example-node",
				"node-role":    "primary",
			}},
	}
	for _, tt := range table {
		actual := newLabels(tt.clusterName, tt.nodeName, tt.primary)
		eq := reflect.DeepEqual(actual, tt.expected)
		if !eq {
			t.Errorf("Test failed, expected: '%v', got: '%v'", tt.expected, actual)
		}
	}
}

func TestNewResourceRequirements(t *testing.T) {
	lCPU, _ := resource.ParseQuantity(defaultCPULimit)
	lMem, _ := resource.ParseQuantity(defaultMemoryLimit)

	rCPU, _ := resource.ParseQuantity(defaultCPURequest)
	rMem, _ := resource.ParseQuantity(defaultMemoryRequest)

	lCPUCustom, _ := resource.ParseQuantity("2000m")
	lMemCustom, _ := resource.ParseQuantity("500Mi")

	rCPUCustom, _ := resource.ParseQuantity("50m")
	rMemCustom, _ := resource.ParseQuantity("250Mi")

	table := []struct {
		resRequirements corev1.ResourceRequirements
		expected        corev1.ResourceRequirements
	}{
		{
			corev1.ResourceRequirements{},
			corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    lCPU,
					corev1.ResourceMemory: lMem,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    rCPU,
					corev1.ResourceMemory: rMem,
				},
			},
		},
		{
			corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    lCPUCustom,
					corev1.ResourceMemory: lMemCustom,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    rCPUCustom,
					corev1.ResourceMemory: rMemCustom,
				},
			},

			corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    lCPUCustom,
					corev1.ResourceMemory: lMemCustom,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    rCPUCustom,
					corev1.ResourceMemory: rMemCustom,
				},
			},
		},
	}
	for _, tt := range table {
		actual := newResourceRequirements(tt.resRequirements)
		eq := reflect.DeepEqual(actual, tt.expected)
		if !eq {
			t.Errorf("Test failed, expected: '%v', got: '%v'", tt.expected, actual)
		}
	}
}
