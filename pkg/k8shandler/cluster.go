package k8shandler

import (
	"context"
	"fmt"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
)

var idSequence int
var nodes map[string]Node
var primaryNode Node

// CreateOrUpdateCluster iterates over all nodes in the current spec
// and ensures cluster reflect desired state
func CreateOrUpdateCluster(request *PostgreSQLRequest) (bool, error) {
	var err error
	var requeue = false
	var repmgrClusterUp = true // flag to track whether all nodes are registered to repmgr cluster

	if nodes == nil {
		nodes = make(map[string]Node)
	}
	if request.cluster.Status.Nodes == nil {
		request.cluster.Status.Nodes = make(map[string]postgresqlv1.PostgreSQLNodeStatus)
	}
	if primaryNode == nil {
		if err := createPrimaryNode(request); err != nil {
			return false, err
		}
	}
	// Loop over all nodes listed in the spec
	for name, specNode := range request.cluster.Spec.Nodes {
		requeue, err = createOrUpdateNode(request, name, &specNode)
		if err != nil {
			log.Error(err, "Non-critical issue")
		}
		node, _ := nodes[name]
		if node.isReady() {
			if err := updateNodeStatus(request, node); err != nil {
				log.Error(err, "Non-critical issue")
				//return false, err
			}
		} else {
			repmgrClusterUp = false
		}
	}
	if err := deleteExtraNodes(request); err != nil {
		log.Error(err, "Non-critical issue")
		//	return false, err
	}
	log.Info(fmt.Sprintf("Nodes after update: %v", nodes))
	if !repmgrClusterUp {
		return true, nil
	}
	return requeue, nil
}

func getHighestPriority(nodeMap map[string]postgresqlv1.PostgreSQLNode) (string, *postgresqlv1.PostgreSQLNode, error) {
	var highestPriority int
	var highestName string

	if len(nodeMap) == 0 {
		return "", nil, fmt.Errorf("Empty map, cannot choose a primary node")
	}

	for name, node := range nodeMap {
		if highestName == "" {
			highestName = name
			highestPriority = node.Priority
		} else if node.Priority > highestPriority {
			highestName = name
			highestPriority = node.Priority
		}
	}
	node := nodeMap[highestName]
	return highestName, &node, nil
}

// createNode creates a new node, asigns id to it and adds it to the nodes map
func createNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode, primary bool) (Node, error) {
	idSequence++
	node := newDeploymentNode(request, name, specNode, idSequence, primary)
	if err := node.create(request); err != nil {
		return nil, err
	}
	nodes[name] = node
	return node, nil
}

// createPrimaryNode creates primary node if it doesn't exists
// TODO handle lost primary reference
func createPrimaryNode(request *PostgreSQLRequest) error {
	name, specNode, err := getHighestPriority(request.cluster.Spec.Nodes)
	log.Info(fmt.Sprintf("Creating new primary node %v", name))
	if err != nil {
		return fmt.Errorf("Nodes spec is empty, cannot choose master node")
	}
	node, err := createNode(request, name, specNode, true)
	if err != nil {
		return err
	}
	primaryNode = node
	return nil
}

// createOrUpdateNode creates a node in case it's not present in nodes map, updates the existing one
// otherwise
func createOrUpdateNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode) (bool, error) {
	var requeue = false
	node, ok := nodes[name]
	if ok {
		// Update existing node
		requeue, err := node.update(request, specNode, primaryNode.dbClient())
		if err != nil {
			return requeue, err
		}
	} else {
		// Create a new node
		_, err := createNode(request, name, specNode, false)
		if err != nil {
			return requeue, err
		}
	}
	return requeue, nil
}

// updateNodeStatus asigns current status of the node to status map
func updateNodeStatus(request *PostgreSQLRequest, node Node) error {
	request.cluster.Status.Nodes[node.name()] = node.status()
	if err := request.client.Status().Update(context.TODO(), request.cluster); err != nil {
		return fmt.Errorf("Failed to update status of PostgreSQL node %v: %v", node.name(), err)
	}
	return nil
}

// deleteExtraNodes deletes all nodes which are not listed in current spec
func deleteExtraNodes(request *PostgreSQLRequest) error {
	for name, deployedNode := range nodes {
		_, ok := request.cluster.Spec.Nodes[name]
		if !ok {
			log.Info(fmt.Sprintf("Deleting node %v", name))
			if err := deployedNode.delete(request); err != nil {
				return err
			}
			delete(nodes, name)
		}
	}
	return nil
}
