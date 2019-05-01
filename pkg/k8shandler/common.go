package k8shandler

func NewLabels(clusterName, nodeName string) map[string]string {
	return map[string]string{
		"cluster-name": clusterName,
		"node-name":    nodeName,
	}
}
