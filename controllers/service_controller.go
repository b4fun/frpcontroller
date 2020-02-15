package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	frpv1 "github.com/b4fun/frpcontroller/api/v1"
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

	var service frpv1.Service
	err := r.Get(ctx, req.NamespacedName, &service)
	switch {
	case err == nil:
		return r.handleCreateOrUpdate(ctx, logger, service)
	case apierrors.IsNotFound(err):
		return r.handleDeleted(ctx, logger, service)
	default:
		logger.Error(err, "get service failed")

		return ctrl.Result{}, err
	}
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&frpv1.Service{}).
		Complete(r)
}

func (r *ServiceReconciler) handleCreateOrUpdate(
	ctx context.Context,
	logger logr.Logger,
	service frpv1.Service,
) (ctrl.Result, error) {
	logger.Info(fmt.Sprintf("to find endpoint: %s", service.Spec.Endpoint))

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) handleDeleted(
	ctx context.Context,
	logger logr.Logger,
	service frpv1.Service,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
