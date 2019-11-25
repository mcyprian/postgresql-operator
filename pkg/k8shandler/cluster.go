package k8shandler

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
)

var idSequence int
var nodes map[string]Node
var primaryNode Node

const (
	// PrimaryRegister registers primary node into the cluster
	PrimaryRegister = "primary register"
	// StandbyRegister registers standby node into the cluster
	StandbyRegister = "standby register"
	// NodeRejoin rejoins the node, which was previously deleted
	NodeRejoin = "node rejoin"
)

// CreateOrUpdateCluster iterates over all nodes in the current spec
// and ensures cluster reflect desired state
func (request *PostgreSQLRequest) CreateOrUpdateCluster() (bool, error) {
	var err error
	var requeue = false
	var status postgresqlv1.PostgreSQLNodeStatus
	var repmgrClusterUp = true // flag to track whether all nodes are registered to repmgr cluster

	if nodes == nil {
		nodes = make(map[string]Node)
	}
	getNodes(request) // search for lost references of nodes in the cluster

	if request.cluster.Status.Nodes == nil {
		request.cluster.Status.Nodes = make(map[string]postgresqlv1.PostgreSQLNodeStatus)
	}
	if primaryNode == nil {
		primaryNode, err = getPrimaryNode()
		if err != nil {
			if err := createPrimaryNode(request); err != nil {
				return false, err
			}
		}
	}
	log.Info("Running create or update for primary service")
	err = request.CreateOrUpdateService("postgresql-primary", primaryNode.name())
	if err != nil {
		log.Error(err, "Failed to create or update primary Service")
		return true, err
	}
	clusterStatus := request.cluster.Status.DeepCopy()
	// Loop over all nodes listed in the spec
	for name, specNode := range request.cluster.Spec.Nodes {
		node, ok := nodes[name]
		if ok {
			if node.isReady() {
				status = node.status()
				clusterStatus.Nodes[node.name()] = status
				if status.Role == postgresqlv1.PostgreSQLNodeRolePrimary && name != primaryNode.name() {
					log.Info(fmt.Sprintf("Failover detected: the new primary node is %v", name))
					primaryNode = node
					log.Info(fmt.Sprintf("Updating primary service selector to %v", primaryNode.name()))
					err = request.CreateOrUpdateService("postgresql-primary", primaryNode.name())
					if err != nil {
						log.Error(err, "Failed to create or update primary Service")
						return true, err
					}
				}
			} else {
				repmgrClusterUp = false
			}
		} else {
			repmgrClusterUp = false
		}
		requeue, err = createOrUpdateNode(request, name, &specNode)
		if err != nil {
			log.Error(err, "Non-critical issue")
			repmgrClusterUp = false
		}
	}
	if err := deleteExtraNodes(request, clusterStatus); err != nil {
		log.Error(err, "Non-critical issue")
	}
	log.Info(fmt.Sprintf("Nodes after update: %v", nodes))

	if !repmgrClusterUp {
		return true, nil
	}
	if err := UpdateClusterStatus(request, clusterStatus); err != nil {
		log.Error(err, "Non-critical issue")
	}

	return requeue, nil
}

func getHighestPriority(nodeMap map[string]postgresqlv1.PostgreSQLNode) (string, *postgresqlv1.PostgreSQLNode, error) {
	var highestPriority int
	var highestName string

	if len(nodeMap) == 0 {
		return "", nil, fmt.Errorf("Empty map, cannot choose a primary node")
	}
	keys := make([]string, len(nodeMap))
	for k := range nodeMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		node := nodeMap[name]
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

// getNodes scans the cluster and adds all existing nodes to the nodes map
func getNodes(request *PostgreSQLRequest) {

	listOpts := client.MatchingLabels(newLabels(request.cluster.Name, ""))
	deploymentList := &appsv1.DeploymentList{}
	err := request.client.List(context.TODO(), listOpts, deploymentList)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to retrieve list of deployments for cluster %v", request.cluster.Name))
	}
	for _, deployment := range deploymentList.Items {
		_, ok := nodes[deployment.ObjectMeta.Name]
		if !ok {
			repmgrPassword, err := getRepmgrPassword(request)
			if err != nil {
				log.Error(err, "Failed to retrieve Repmgr password.")
				return
			}
			log.Info(fmt.Sprintf("Attaching existing deployment: %v", deployment.ObjectMeta.Name))
			nodes[deployment.ObjectMeta.Name] = attachDeploymentNode(request, deployment.ObjectMeta.Name, &deployment, repmgrPassword)
		}
	}
}

// getPrimaryNode searches for primary node in nodes map
func getPrimaryNode() (Node, error) {
	for name, node := range nodes {
		status := node.status()
		log.Info(fmt.Sprintf("Node %v role %v.", name, status.Role))
		if status.Role == postgresqlv1.PostgreSQLNodeRolePrimary {
			log.Info(fmt.Sprintf("Lost primary node %v discovered.", name))
			return node, nil
		}
	}
	return nil, fmt.Errorf("Primary node not found in the cluster.")
}

// getRepmgrPassword retrieves Repmgr password from the secret
func getRepmgrPassword(request *PostgreSQLRequest) (string, error) {
	secretData, err := extractSecret(request.cluster.Name, request.cluster.Namespace, request.client)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to extract secret %v", request.cluster.Name))
		return "", err
	}
	repmgrPassword, ok := secretData["repmgr-password"]
	if !ok {
		log.Error(err, fmt.Sprintf("Repmgr password not found in secret %v", request.cluster.Name))
		return "", err
	}
	return string(repmgrPassword), nil
}

// createNode creates a new node, asigns an id to it and adds it to the nodes map
func createNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode, operation string) (Node, error) {
	var id = -1
	// Try to get existing id
	if primaryNode != nil {
		db := primaryNode.dbClient()
		info := db.getNodeInfo(name)
		if err := db.err(); err == nil {
			id = info.id
			operation = NodeRejoin
		}
	}
	// increment sequence if id was not obtained successfully
	if id == -1 {
		idSequence++
		id = idSequence
	}
	repmgrPassword, err := getRepmgrPassword(request)
	if err != nil {
		return nil, err
	}

	node := newDeploymentNode(request, name, specNode, id, repmgrPassword, operation)
	if err := node.create(request); err != nil {
		return nil, err
	}
	nodes[name] = node
	return node, nil
}

// createPrimaryNode creates primary node if it doesn't exists
func createPrimaryNode(request *PostgreSQLRequest) error {
	name, specNode, err := getHighestPriority(request.cluster.Spec.Nodes)
	log.Info(fmt.Sprintf("Creating new primary node %v", name))
	if err != nil {
		return fmt.Errorf("Nodes spec is empty, cannot choose master node")
	}
	node, err := createNode(request, name, specNode, PrimaryRegister)
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
		_, err := createNode(request, name, specNode, StandbyRegister)
		if err != nil {
			return requeue, err
		}
	}
	return requeue, nil
}

// deleteExtraNodes deletes all nodes which are not listed in current spec
func deleteExtraNodes(request *PostgreSQLRequest, clusterStatus *postgresqlv1.PostgreSQLStatus) error {
	for name, deployedNode := range nodes {
		_, ok := request.cluster.Spec.Nodes[name]
		if !ok {
			log.Info(fmt.Sprintf("Deleting node %v", name))
			if err := deployedNode.delete(request); err != nil {
				return err
			}
			delete(nodes, name)
			delete(clusterStatus.Nodes, name)
		}
	}
	return nil
}
