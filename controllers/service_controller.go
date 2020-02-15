package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	frpv1 "github.com/b4fun/frpcontroller/api/v1"
)

const (
	serviceOwnerKey = ".metadata.controller"
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
		return r.handleCreateOrUpdate(ctx, logger, &service)
	case apierrors.IsNotFound(err):
		return r.handleDeleted(ctx, logger, &service)
	default:
		logger.Error(err, "get service failed")

		return ctrl.Result{}, err
	}
}

func (r *ServiceReconciler) handleCreateOrUpdate(
	ctx context.Context,
	logger logr.Logger,
	service *frpv1.Service,
) (ctrl.Result, error) {
	endpointName := client.ObjectKey{
		Name:      service.Spec.Endpoint,
		Namespace: service.Namespace,
	}

	if service.Labels == nil {
		service.Labels = map[string]string{}
	}
	if v, exists := service.Labels[labelKeyEndpointName]; !exists || v != endpointName.Name {
		service.Labels[labelKeyEndpointName] = endpointName.Name
		if err := r.Update(ctx, service); err != nil {
			logger.Error(err, "update labels failed")
			return ctrl.Result{}, err
		}
	}

	var (
		kserviceList  corev1.ServiceList
		kserviceBound *corev1.Service
	)
	err := r.List(
		ctx, &kserviceList,
		client.InNamespace(service.Namespace),
		client.MatchingFields{serviceOwnerKey: service.Name},
	)
	if err != nil {
		logger.Error(err, "list services failed")
		return ctrl.Result{}, err
	}
	for _, kservice := range kserviceList.Items {
		kservice.Spec.Selector = service.Spec.Selector
		kservice.Spec.Ports = nil
		for _, port := range service.Spec.Ports {
			kservice.Spec.Ports = append(kservice.Spec.Ports, port.ToCorev1ServicePort())
		}
		err = r.Update(ctx, &kservice)
		if err != nil {
			logger.Error(err, fmt.Sprintf("update corev1.service %s failed", service.Name))
			return ctrl.Result{}, err
		}
		logger.Info(fmt.Sprintf("updated corev1.service: %s", service.Name))
		kserviceBound = &kservice
	}
	if kserviceBound == nil {
		var kservicePorts []corev1.ServicePort
		for _, port := range service.Spec.Ports {
			kservicePorts = append(kservicePorts, port.ToCorev1ServicePort())
		}
		kserviceBound = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-frpc-", service.Name),
				Namespace:    service.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeClusterIP,
				Selector: service.Spec.Selector,
				Ports:    kservicePorts,
			},
		}
		err = ctrl.SetControllerReference(service, kserviceBound, r.Scheme)
		if err != nil {
			logger.Error(err, "set controller reference failed")
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, kserviceBound)
		if err != nil {
			logger.Error(err, "create corev1.service failed")
			return ctrl.Result{}, err
		}
		logger.Info(fmt.Sprintf("created service %s", kserviceBound.Name))
	}
	if kserviceBound.Spec.ClusterIP != "" {
		if service.Annotations == nil {
			service.Annotations = map[string]string{}
		}
		service.Annotations[annotationKeyServiceClusterIP] = kserviceBound.Spec.ClusterIP
		err = r.Update(ctx, service)
		if err != nil {
			logger.Error(err, "update service failed")
			return ctrl.Result{}, err
		}
		logger.Info(fmt.Sprintf(
			"binded cluster ip %s to service: %s",
			kserviceBound.Spec.ClusterIP, service.Name,
		))
	}

	var endpoint frpv1.Endpoint
	err = r.Get(ctx, endpointName, &endpoint)
	switch {
	case err == nil:
		// TODO: read config & update status
	case apierrors.IsNotFound(err):
		logger.Info(fmt.Sprintf("endpoint %s does not exist, try later", endpointName.Name))

		service.Status.State = frpv1.ServiceStateInactive
		if err := r.Status().Update(ctx, service); err != nil {
			logger.Error(err, "update service status failed")
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: time.Duration(10 * time.Second),
		}, nil
	default:
		logger.Error(err, "get endpoint failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: time.Duration(10 * time.Second),
	}, nil
}

func (r *ServiceReconciler) handleDeleted(
	ctx context.Context,
	logger logr.Logger,
	service *frpv1.Service,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(
		&corev1.Service{}, serviceOwnerKey,
		func(rawObj runtime.Object) []string {
			kservice := rawObj.(*corev1.Service)
			owner := metav1.GetControllerOf(kservice)
			if owner == nil {
				return nil
			}
			if owner.APIVersion != apiGVStr || owner.Kind != KindService {
				return nil
			}
			return []string{owner.Name}
		},
	)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&frpv1.Service{}).
		Complete(r)
}
