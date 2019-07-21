package k8shandler

import (
	"context"
	"fmt"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type deploymentNode struct {
	self *appsv1.Deployment
	svc  *corev1.Service
	db   *database
}

func newDeploymentNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode, nodeId int, primary bool) *deploymentNode {
	return &deploymentNode{
		self: newDeployment(request, name, specNode, nodeId, primary),
		svc:  newClusterIPService(request, name, false),
		db:   newRepmgrDatabase(name),
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
	if err := CreateOrUpdateService(request, node.svc.ObjectMeta.Name, false); err != nil {
		return fmt.Errorf("Failed to create service resource %v", err)
	}
	node.db.initialize()
	if err := node.db.err(); err != nil {
		return fmt.Errorf("Failed to initialize repmgr database connection %v", err)
	}

	return nil
}

func (node *deploymentNode) update(request *PostgreSQLRequest, specNode *postgresqlv1.PostgreSQLNode, writableDB *database) (bool, error) {
	// TODO update node to reflect spec
	if err := CreateOrUpdateService(request, node.svc.ObjectMeta.Name, false); err != nil {
		return false, fmt.Errorf("Failed to create service resource %v", err)
	}
	current := node.self.DeepCopy()
	if err := request.client.Get(context.TODO(), types.NamespacedName{Name: node.name(), Namespace: request.cluster.Namespace}, current); err != nil {
		return false, fmt.Errorf("Failed to get deployment %v: %v", node.name(), err)
	}
	// Update labels if role was changed inside repmgr
	if node.isReady() {
		role, priority := node.db.getNodeInfo(node.name())
		if err := node.db.err(); err != nil {
			log.Error(err, fmt.Sprintf("Failed to query role of node %v", node.name()))
		} else {
			if priority != specNode.Priority {
				writableDB.updateNodePriority(node.name(), specNode.Priority)
				if err := node.db.err(); err != nil {
					log.Error(err, fmt.Sprintf("Failed to update priority of node %v", node.name()))
				}
			}
		}

		if err := request.client.Update(context.TODO(), current); err != nil {
			return false, fmt.Errorf("Failed to update deployment %v: %v", node.name(), err)
		}
	}
	node.self = current
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
	role, priority := node.db.getNodeInfo(node.name())
	status := postgresqlv1.PostgreSQLNodeStatus{
		DeploymentName: node.self.ObjectMeta.Name,
		ServiceName:    node.svc.ObjectMeta.Name,
		PgVersion:      node.db.version(),
		Role:           role,
		Priority:       priority,
	}
	if err := node.db.err(); err != nil {
		log.Error(err, "Failed to get node info")
	}
	return status
}

func (node *deploymentNode) dbClient() *database {
	return node.db
}

// getPod returns pod which was created by the node
func (node *deploymentNode) getPod(request *PostgreSQLRequest) (corev1.Pod, error) {
	podList := corev1.PodList{}
	labelSelector := labels.SelectorFromSet(newLabels(request.cluster.Name, node.name(), false))
	listOps := &client.ListOptions{Namespace: request.cluster.Namespace, LabelSelector: labelSelector}
	err := request.client.List(context.TODO(), listOps, &podList)
	if err != nil || len(podList.Items) < 1 {
		return corev1.Pod{}, fmt.Errorf("Failed to get pods for node %v: %v", node.name(), err)
	}
	return podList.Items[0], nil
}

func (node *deploymentNode) isReady() bool {
	return node.self.Status.ReadyReplicas == 1
}

func (node *deploymentNode) isRegistered(request *PostgreSQLRequest) (bool, error) {
	result := node.db.isRegistered(node.name())
	if err := node.db.err(); err != nil {
		return false, fmt.Errorf("Failed to check node %v register status: %v", node.name(), err)
	}
	return result, nil
}

func (node *deploymentNode) register(request *PostgreSQLRequest) error {
	var primary bool = false
	role, _ := node.self.ObjectMeta.Labels["node-role"]
	if role == "primary" {
		primary = true
	}
	pod, err := node.getPod(request)
	if err != nil {
		return err
	}
	return repmgrRegister(request, pod, primary)
}
