package k8shandler

import (
	"context"
	"fmt"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type deploymentNode struct {
	self        *appsv1.Deployment
	svc         *corev1.Service
	db          *database
	initPrimary bool
}

func newDeploymentNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode, primary bool) *deploymentNode {
	return &deploymentNode{
		self:        newDeployment(request, name, specNode, primary),
		svc:         newClusterIPService(request, name, primary),
		db:          newRepmgrDatabase(name),
		initPrimary: primary,
	}
}

func (node *deploymentNode) name() string {
	return node.self.ObjectMeta.Name
}

func (node *deploymentNode) create(request *PostgreSQLRequest) error {
	if err := request.client.Create(context.TODO(), node.self); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failed to create node resource %v", err)
		}
	}
	if err := CreateOrUpdateService(request, node.svc.ObjectMeta.Name, node.initPrimary); err != nil {
		return fmt.Errorf("Failed to create service resource %v", err)
	}
	if err := node.db.initialize(); err != nil {
		return fmt.Errorf("Failed to initialize repmgr database connection %v", err)
	}

	return nil
}

func (node *deploymentNode) update(request *PostgreSQLRequest, specNode *postgresqlv1.PostgreSQLNode) (bool, error) {
	// TODO update node to reflect spec
	//if err := CreateOrUpdateService(request, node.svc); err != nil {
	//	return fmt.Errorf("Failed to create service resource %v", err)
	//}
	return false, nil
}

func (node *deploymentNode) delete(request *PostgreSQLRequest) error {
	if err := request.client.Delete(context.TODO(), node.self); err != nil {
		return fmt.Errorf("Failed to delete node resource %v", err)
	}
	if err := request.client.Delete(context.TODO(), node.svc); err != nil {
		return fmt.Errorf("Failed to delete service resource %v", err)
	}
	node.db.engine.Close()
	return nil
}

func (node *deploymentNode) status() postgresqlv1.PostgreSQLNodeStatus {
	// TODO Return node role and status
	version, err := node.db.version()
	if err != nil {
		log.Error(err, "Failed to execute SQL query")
	}

	return postgresqlv1.PostgreSQLNodeStatus{
		DeploymentName: node.self.ObjectMeta.Name,
		ServiceName:    node.svc.ObjectMeta.Name,
		PgVersion:      version,
	}
}

// getPod returns pod which was created by the node
func (node *deploymentNode) getPod(request *PostgreSQLRequest) (corev1.Pod, error) {
	podList := corev1.PodList{}
	labelSelector := labels.SelectorFromSet(newLabels(request.cluster.Name, node.self.ObjectMeta.Name, false))
	listOps := &client.ListOptions{Namespace: request.cluster.Namespace, LabelSelector: labelSelector}
	err := request.client.List(context.TODO(), listOps, &podList)
	if err != nil || len(podList.Items) < 1 {
		return corev1.Pod{}, fmt.Errorf("Failed to get pods for node %v: %v", node.self.ObjectMeta.Name, err)
	}
	return podList.Items[0], nil
}

func (node *deploymentNode) isReady(request *PostgreSQLRequest) (bool, error) {
	pod, err := node.getPod(request)
	if err != nil {
		return false, err
	}
	return isReady(pod), nil
}

func (node *deploymentNode) isRegistered(request *PostgreSQLRequest) (bool, error) {
	pod, err := node.getPod(request)
	if err != nil {
		return false, err
	}
	return isRegistered(request, pod)
}

func (node *deploymentNode) register(request *PostgreSQLRequest) error {
	pod, err := node.getPod(request)
	if err != nil {
		return err
	}
	return repmgrRegister(request, pod)
}
