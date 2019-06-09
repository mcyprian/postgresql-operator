package k8shandler

func newLabels(clusterName, nodeName string) map[string]string {
	return map[string]string{
		"cluster-name": clusterName,
		"node-name":    nodeName,
	}
}

func selectorForPg(clusterName string) map[string]string {

	return map[string]string{
		"cluster-name": clusterName,
	}
}
