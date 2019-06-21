package k8shandler

import (
	"fmt"
)

var nodes map[string]Node

// CreateOrUpdateCluster iterates over all nodes in the current spec
// and ensures cluster reflect desired state
func CreateOrUpdateCluster(request *PostgreSQLRequest) (bool, error) {
	var requeue bool
	var err error

	if nodes == nil {
		nodes = make(map[string]Node)
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
			node = newDeploymentNode(request, name, &specNode)
			err := node.create(request)
			if err != nil {
				return false, err
			}
			nodes[name] = node
		}
	}
	// Delete all extra nodes
	for name, deployedNode := range nodes {
		_, ok := request.cluster.Spec.Nodes[name]
		if !ok {
			log.Info(fmt.Sprintf("Deleting node %v", name))
			err := deployedNode.delete(request)
			if err != nil {
				return false, err
			}
			delete(nodes, name)
		}
	}
	log.Info(fmt.Sprintf("Nodes after update %v", nodes))
	return requeue, nil
}
