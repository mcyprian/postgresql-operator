package k8shandler

import (
	"context"
	"fmt"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

type deploymentNode struct {
	self *appsv1.Deployment
	svc  *corev1.Service
	db   *database
}

func newDeploymentNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode, nodeID int, repmgrPassword string, operation string) *deploymentNode {
	return &deploymentNode{
		self: newDeployment(request, name, specNode, nodeID, operation),
		svc:  newService(request, name, name),
		db:   newRepmgrDatabase(name, repmgrPassword),
	}
}

func attachDeploymentNode(request *PostgreSQLRequest, name string, deployment *appsv1.Deployment, repmgrPassword string) *deploymentNode {
	node := &deploymentNode{
		self: deployment,
		svc:  newService(request, name, name),
		db:   newRepmgrDatabase(name, repmgrPassword),
	}
	node.db.initialize()
	if err := node.db.err(); err != nil {
		fmt.Errorf("Failed to initialize repmgr database connection %v", err)
	}
	return node
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
	if err := request.CreateOrUpdateService(node.svc.ObjectMeta.Name, node.svc.ObjectMeta.Name); err != nil {
		return fmt.Errorf("Failed to create service resource %v", err)
	}
	node.db.initialize()
	if err := node.db.err(); err != nil {
		return fmt.Errorf("Failed to initialize repmgr database connection %v", err)
	}

	return nil
}

func (node *deploymentNode) update(request *PostgreSQLRequest, specNode *postgresqlv1.PostgreSQLNode, writableDB *database) (bool, error) {
	if err := request.CreateOrUpdateService(node.svc.ObjectMeta.Name, node.svc.ObjectMeta.Name); err != nil {
		return false, fmt.Errorf("Failed to create service resource %v", err)
	}
	current := node.self.DeepCopy()
	if err := request.client.Get(context.TODO(), types.NamespacedName{Name: node.name(), Namespace: request.cluster.Namespace}, current); err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("CREATING LOST DEPLOYMENT %v", node.self.ObjectMeta.Name))
			nodeInfo := writableDB.getNodeInfo(node.name())
			node.self = newDeployment(request, node.name(), specNode, nodeInfo.id, NodeRejoin)
			if err := request.client.Create(context.TODO(), node.self); err != nil {
				return true, fmt.Errorf("Failed to create node resource %v", err)
			}
			return false, nil
		} else {
			return false, fmt.Errorf("Failed to get deployment %v: %v", node.name(), err)
		}
	}
	current.Spec.Template.Spec.Containers[0].Resources = newResourceRequirements(specNode.Resources)
	current.Spec.Template.Spec.Volumes[0] = newVolume(request, node.name(), &specNode.Storage)
	current.Spec.Template.Spec.Containers[0].Image = specNode.Image

	if node.isReady() {
		info := node.db.getNodeInfo(node.name())
		if err := node.db.err(); err != nil {
			log.Error(err, fmt.Sprintf("Failed to query role of node %v", node.name()))
		} else {
			if info.priority != specNode.Priority {
				writableDB.updateNodePriority(node.name(), specNode.Priority)
				if err := node.db.err(); err != nil {
					log.Error(err, fmt.Sprintf("Failed to update priority of node %v", node.name()))
				}
			}
		}
		if err := request.client.Update(context.TODO(), current); err != nil {
			return true, fmt.Errorf("Failed to update deployment %v: %v", node.name(), err)
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
	info := node.db.getNodeInfo(node.name())
	status := postgresqlv1.PostgreSQLNodeStatus{
		DeploymentName: node.self.ObjectMeta.Name,
		ServiceName:    node.svc.ObjectMeta.Name,
		PgVersion:      node.db.version(),
		Role:           info.role,
		Priority:       info.priority,
	}
	if err := node.db.err(); err != nil {
		log.Error(err, "Failed to get node info")
	}
	return status
}

func (node *deploymentNode) dbClient() *database {
	return node.db
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
