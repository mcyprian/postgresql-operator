package postgresql

import (
	"context"
	"reflect"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	k8shander "github.com/mcyprian/postgresql-operator/pkg/k8shandler"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_postgresql")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PostgreSQL Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePostgreSQL{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("postgresql-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PostgreSQL
	err = c.Watch(&source.Kind{Type: &postgresqlv1.PostgreSQL{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner PostgreSQL
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &postgresqlv1.PostgreSQL{},
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
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PostgreSQL object and makes changes based on the state read
// and what is in the PostgreSQL.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePostgreSQL) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling PostgreSQL")

	// Fetch the PostgreSQL instance
	postgresql := &postgresqlv1.PostgreSQL{}
	err := r.client.Get(context.TODO(), request.NamespacedName, postgresql)
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

	reqLogger.Info("Running create or update for Service")
	err = k8shander.CreateOrUpdateService(postgresql, r.client)
	if err != nil {
		reqLogger.Info("Failed to create of update Service")
		return reconcile.Result{}, err
	}

	found := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: postgresql.Name, Namespace: postgresql.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new set
		set := k8shander.NewStatefulSet(postgresql, r.scheme)
		reqLogger.Info("Creating a new StatefulSet", "StatefulSet.Namespace", set.Namespace, "StatefulSet.Name", set.Name)
		err = r.client.Create(context.TODO(), set)
		if err != nil {
			reqLogger.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", set.Namespace, "StatefulSet.Name", set.Name)
			return reconcile.Result{}, err
		}
		// StatefulSet created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get StatefulSet")
		return reconcile.Result{}, err
	}

	// Ensure the set size is the same as the spec
	size := postgresql.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update StatefulSet", "StatefulSet.Namespace", found.Namespace, "StatefulSet.Name", found.Name)
			return reconcile.Result{}, err
		}
	}
	// Spec updated - return and requeue
	return reconcile.Result{Requeue: true}, nil

	// Define a new Pod object
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(k8shander.NewLabels("postgresql", postgresql.Name))
	listOps := &client.ListOptions{Namespace: postgresql.Namespace, LabelSelector: labelSelector}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "PostgreSQL.Namespace", postgresql.Namespace, "PostgreSQL.Name", postgresql.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, postgresql.Status.Nodes) {
		postgresql.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), postgresql)
		if err != nil {
			reqLogger.Error(err, "Failed to update PostgreSQL status")
			return reconcile.Result{}, err
		}
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
