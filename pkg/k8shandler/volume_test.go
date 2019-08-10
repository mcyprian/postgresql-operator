package k8shandler

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestNewVolume(t *testing.T) {
	testSize, _ := resource.ParseQuantity("100Mi")
	testRequest := PostgreSQLRequest{
		cluster: &postgresqlv1.PostgreSQL{},
		scheme:  &runtime.Scheme{},
	}
	testRequest.cluster.Name = "test-cluster"

	table := []struct {
		specVol  postgresqlv1.PostgreSQLStorageSpec
		expected corev1.Volume
	}{
		{
			postgresqlv1.PostgreSQLStorageSpec{
				StorageClassName: nil,
				Size:             nil,
			},
			corev1.Volume{
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			postgresqlv1.PostgreSQLStorageSpec{
				StorageClassName: nil,
				Size:             &testSize,
			},
			corev1.Volume{
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: &testSize,
					},
				},
			},
		},
	}
	for _, tt := range table {
		actual := newVolume(&testRequest, "test", &tt.specVol)
		eq := reflect.DeepEqual(actual.VolumeSource, tt.expected.VolumeSource)
		if !eq {
			t.Errorf("Test failed, expected: '%v', got: '%v'", tt.expected, actual)
		}
	}
}
