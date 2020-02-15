package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

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
	ctx := context.Background()
	logger := r.Log.WithValues("service", req.NamespacedName)

	logger.Info(fmt.Sprintf("namespaced name: %v", req.NamespacedName))

	var service corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		logger.Error(err, "unable to fetch frp service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info(fmt.Sprintf("loaded service: %v, %t", service.Spec, service.Status.Active))

	return ctrl.Result{
		RequeueAfter: time.Duration(5) * time.Second,
	}, nil
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
