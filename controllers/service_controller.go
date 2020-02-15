package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "github.com/b4fun/frpcontroller/api/v1"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.go.build4.fun,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.go.build4.fun,resources=services/status,verbs=get;update;patch

func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("service", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
