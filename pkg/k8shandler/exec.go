package k8shandler

import (
	"bytes"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecToPodThroughAPI uninterractively exec to the pod with the command specified.
func ExecToPodThroughAPI(config *rest.Config, clientset *kubernetes.Clientset, command []string, containerName, podName, namespace string, stdin io.Reader) (string, string, error) {
	reqLogger := log.WithValues("Request.Namespace", namespace)
	req := clientset.Core().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		reqLogger.Error(err, "Failed to register scheme.")
		return "", "", err
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	reqLogger.Info(fmt.Sprintf("Request URL: %v", req.URL().String()))

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		reqLogger.Error(err, "Remote command execution failed.")
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		reqLogger.Error(err, "Streaming exec output failed.")
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}
