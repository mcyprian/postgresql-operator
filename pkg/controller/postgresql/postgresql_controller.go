package postgresql

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	api "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	k8shander "github.com/mcyprian/postgresql-operator/pkg/k8shandler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_postgresql")

// Add creates a new PostgreSQL Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	var clientset *kubernetes.Clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err, "Failed to create rest config")
	} else {
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Error(err, "Failed to create clientset")
		}
	}

	return &ReconcilePostgreSQL{client: mgr.GetClient(), scheme: mgr.GetScheme(), restConfig: config, clientset: clientset}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("postgresql-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PostgreSQL
	err = c.Watch(&source.Kind{Type: &api.PostgreSQL{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner PostgreSQL
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &api.PostgreSQL{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePostgreSQL{}

// ReconcilePostgreSQL reconciles a PostgreSQL object
type ReconcilePostgreSQL struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	scheme     *runtime.Scheme
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
}

// Reconcile reads that state of the cluster for a PostgreSQL object and makes changes based on the state read
// and what is in the PostgreSQL.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePostgreSQL) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling PostgreSQL")

	// Fetch the PostgreSQL instance
	p := &api.PostgreSQL{}
	err := r.client.Get(context.TODO(), request.NamespacedName, p)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	reqLogger.Info("Running create or update for Secret")
	err = k8shander.CreateOrUpdateSecret(p, r.client)
	if err != nil {
		reqLogger.Error(err, "Failed to create of update Secret")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Running create or update for StatefulSet")
	requeue, err := k8shander.CreateOrUpdateStatefulSet(p, r.client, r.scheme)
	if err != nil {
		reqLogger.Error(err, "Failed to create of update StatefulSet")
		return reconcile.Result{}, err
	} else if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	reqLogger.Info("Running create or update for Service")
	err = k8shander.CreateOrUpdateService(p, r.client)
	if err != nil {
		reqLogger.Error(err, "Failed to create of update Service")
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(k8shander.NewLabels("postgresql", p.Name))
	listOps := &client.ListOptions{Namespace: p.Namespace, LabelSelector: labelSelector}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "PostgreSQL.Namespace", p.Namespace, "PostgreSQL.Name", p.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, p.Status.Nodes) {
		p.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), p)
		if err != nil {
			reqLogger.Error(err, "Failed to update PostgreSQL status")
			return reconcile.Result{}, err
		}
	}

	// Register nodes which were not registered to repmgr cluster yet
	repmgrClusterUp := true
	if int32(len(podList.Items)) != p.Spec.Size {
		repmgrClusterUp = false
	}
	for _, pod := range podList.Items {
		if !isReady(pod) {
			repmgrClusterUp = false
		} else {
			registered, _ := isRegistered(p, r.restConfig, r.clientset, pod)
			if !registered {
				err = repmgrRegister(p, r.restConfig, r.clientset, pod)
				if err != nil {
					reqLogger.Error(err, "Repmgr register failed")
					return reconcile.Result{}, err
				}
				repmgrClusterUp = false
			}
		}
	}
	if !repmgrClusterUp {
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// isReady determines whether pod status is Ready
func isReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" && cond.Status == "True" {
			return true
		}
	}
	return false
}

// isRegistered determines whether repmgr node was successfuly registered
func isRegistered(p *api.PostgreSQL, config *rest.Config, clientset *kubernetes.Clientset, pod corev1.Pod) (bool, error) {
	execCommand := []string{"shell-entrypoint", "repmgr", "node", "check"}
	stdout, stderr, err := k8shander.ExecToPodThroughAPI(config, clientset, execCommand, pod.Spec.Containers[0].Name, pod.Name, p.Namespace, nil)
	if err != nil {
		log.Info(fmt.Sprintf("Repmgr node check returned non-zero exit status, stdout: %v, stderr: %v", stdout, stderr))
		return false, err
	} else {
		log.Info("Repmgr node check executed", stdout, stderr)
		if strings.Contains(stdout, "OK") {
			return true, nil
		} else {
			return false, nil
		}
	}
}

func repmgrRegister(p *api.PostgreSQL, config *rest.Config, clientset *kubernetes.Clientset, pod corev1.Pod) error {
	execCommand := []string{"shell-entrypoint", "repmgr-register"}
	stdout, stderr, err := k8shander.ExecToPodThroughAPI(config, clientset, execCommand, pod.Spec.Containers[0].Name, pod.Name, p.Namespace, nil)
	if err != nil {
		log.Error(err, fmt.Sprintf("Repmgr register failed, stdout: %v, stderr: %v", stdout, stderr))
		return err
	} else {
		log.Info("Repmgr register executed", stdout, stderr)
	}
	return nil
}
