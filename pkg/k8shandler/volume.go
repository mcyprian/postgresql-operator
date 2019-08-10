package k8shandler

import (
	"context"
	"fmt"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newVolume(request *PostgreSQLRequest, name string, specVol *postgresqlv1.PostgreSQLStorageSpec) corev1.Volume {
	volSource := corev1.VolumeSource{}

	switch {
	case specVol.StorageClassName != nil && specVol.Size != nil:
		claimName := fmt.Sprintf("%s-%s", request.cluster.Name, name)
		volSource.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: claimName,
		}

		volSpec := corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *specVol.Size,
				},
			},
			//StorageClassName: specVol.StorageClassName,
		}

		err := createPersistentVolumeClaim(request, volSpec, claimName)
		if err != nil {
			log.Error(err, "Unable to create PersistentVolumeClaim")
		}

	case specVol.Size != nil:
		volSource.EmptyDir = &corev1.EmptyDirVolumeSource{
			SizeLimit: specVol.Size,
		}

	default:
		volSource.EmptyDir = &corev1.EmptyDirVolumeSource{}
	}
	return corev1.Volume{
		Name:         name,
		VolumeSource: volSource,
	}
}

func createPersistentVolumeClaim(request *PostgreSQLRequest, pvc corev1.PersistentVolumeClaimSpec, newName string) error {

	claim := newPersistentVolumeClaim(newName, request.cluster.Namespace, pvc)
	controllerutil.SetControllerReference(request.cluster, claim, request.scheme)
	if err := request.client.Create(context.TODO(), claim); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Unable to create PVC: %v", err)
		}
	}

	return nil
}

func newPersistentVolumeClaim(pvcName, namespace string, volSpec corev1.PersistentVolumeClaimSpec) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
	}

	pvc.Spec = volSpec
	return pvc
}
