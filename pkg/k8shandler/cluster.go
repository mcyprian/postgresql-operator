package k8shandler

import (
	"context"
	"fmt"
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
)

var nodes map[string]Node
var primaryNode Node

func getOne(m map[string]postgresqlv1.PostgreSQLNode) (string, *postgresqlv1.PostgreSQLNode, error) {
	for name, node := range m {
		return name, &node, nil
	}
	return "", nil, fmt.Errorf("Empty map, cannot get a key")
}

// CreateOrUpdateCluster iterates over all nodes in the current spec
// and ensures cluster reflect desired state
func CreateOrUpdateCluster(request *PostgreSQLRequest) (bool, error) {
	var requeue bool
	var err error
	var repmgrClusterUp = true // flag to track whether all nodes are registered to repmgr cluster

	if nodes == nil {
		nodes = make(map[string]Node)
	}
	if request.cluster.Status.Nodes == nil {
		request.cluster.Status.Nodes = make(map[string]postgresqlv1.PostgreSQLNodeStatus)
	}
	if primaryNode == nil {
		// Create new primary node TODO: lost primary reference?
		name, specNode, err := getOne(request.cluster.Spec.Nodes)
		if err != nil {
			return false, fmt.Errorf("Nodes spec is empty, cannot choose master node")
		}
		node := newDeploymentNode(request, name, specNode, true)
		if err := node.create(request); err != nil {
			return false, err
		}
		primaryNode = node
		nodes[name] = node
	}
	for name, specNode := range request.cluster.Spec.Nodes {
		node, ok := nodes[name]
		if ok {
			// Update existing node
			requeue, err = node.update(request, &specNode)
			if err != nil {
				return false, err
			}
		} else {
			// Create a new node and add it into nodes map
			node = newDeploymentNode(request, name, &specNode, false)
			if err := node.create(request); err != nil {
				return false, err
			}
			nodes[name] = node
		}
		ready, err := node.isReady(request)
		if err != nil {
			return false, err
		}
		if ready {
			// Update node status
			request.cluster.Status.Nodes[name] = node.status()
			if err := request.client.Status().Update(context.TODO(), request.cluster); err != nil {
				return false, fmt.Errorf("Failed to update PostgreSQL status")
			}
			registered, _ := node.isRegistered(request)
			if !registered {
				// Register to repmgr cluster
				if err := node.register(request); err != nil {
					return true, err
				}
				repmgrClusterUp = false
			}
		} else {
			repmgrClusterUp = false
		}
	}
	// Delete all extra nodes
	for name, deployedNode := range nodes {
		_, ok := request.cluster.Spec.Nodes[name]
		if !ok {
			log.Info(fmt.Sprintf("Deleting node %v", name))
			if err := deployedNode.delete(request); err != nil {
				return false, err
			}
			delete(nodes, name)
		}
	}
	log.Info(fmt.Sprintf("Nodes after update: %v", nodes))
	if !repmgrClusterUp {
		return true, nil
	}
	return requeue, nil
}
