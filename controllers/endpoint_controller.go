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

	frpv1 "github.com/b4fun/frpcontroller/api/v1"
)

const (
	endpointOwnerKey = ".metadata.controller"
)

// EndpointReconciler reconciles a Endpoint object
type EndpointReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=frp.go.build4.fun,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frp.go.build4.fun,resources=endpoints/status,verbs=get;update;patch

func (r *EndpointReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("endpoint", req.NamespacedName)

	var endpoint frpv1.Endpoint
	err := r.Get(ctx, req.NamespacedName, &endpoint)
	switch {
	case err == nil:
		return r.handleCreateOrUpdate(ctx, logger, &endpoint)
	case apierrors.IsNotFound(err):
		return r.handleDeleted(ctx, logger, &endpoint)
	default:
		logger.Error(err, "get endpoint failed")
		return ctrl.Result{}, err
	}
}

func (r *EndpointReconciler) handleCreateOrUpdate(
	ctx context.Context,
	logger logr.Logger,
	endpoint *frpv1.Endpoint,
) (ctrl.Result, error) {
	frpsConfig, err := r.ensureEndpointConfigMap(ctx, logger, endpoint)
	if err != nil {
		return ctrl.Result{}, nil
	}

	_, err = r.ensureEndpointPod(ctx, logger, endpoint, frpsConfig)
	if err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *EndpointReconciler) handleDeleted(
	ctx context.Context,
	logger logr.Logger,
	endpoint *frpv1.Endpoint,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *EndpointReconciler) ensureEndpointConfigMap(
	ctx context.Context,
	logger logr.Logger,
	endpoint *frpv1.Endpoint,
) (*corev1.ConfigMap, error) {
	var (
		frpsConfigList    corev1.ConfigMapList
		frpsConfig        *corev1.ConfigMap
		frpsConfigExisted bool
	)
	err := r.List(
		ctx, &frpsConfigList,
		client.InNamespace(endpoint.Namespace),
		client.MatchingFields{endpointOwnerKey: endpoint.Name},
	)
	if err != nil {
		logger.Error(err, "list endpoint config map failed")
		return nil, err
	}
	if len(frpsConfigList.Items) == 0 {
		logger.Info("no endpoint config map found, will create")
		frpsConfigExisted = false
		frpsConfig = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{},
				Annotations: map[string]string{},
				Name:        fmt.Sprintf("%s-frps", endpoint.Name),
				Namespace:   endpoint.Namespace,
			},
			Data: map[string]string{},
		}
		err := ctrl.SetControllerReference(endpoint, frpsConfig, r.Scheme)
		if err != nil {
			logger.Error(err, "set controller reference failed")
			return nil, err
		}
	} else {
		frpsConfigExisted = true
		frpsConfig = &frpsConfigList.Items[0]
		logger.Info(fmt.Sprintf(
			"found %d config maps, using %s",
			len(frpsConfigList.Items),
			frpsConfig.Name),
		)
	}

	// TODO: generate real config
	if frpsConfig.Data == nil {
		frpsConfig.Data = map[string]string{}
	}
	frpsConfig.Data[frpsFileName] = `
[common]
server_addr = 127.0.0.1
server_port = 1234
token = foobar
`

	if frpsConfigExisted {
		if err := r.Update(ctx, frpsConfig); err != nil {
			logger.Error(err, "update config map failed")
			return nil, err
		}
	} else {
		if err := r.Create(ctx, frpsConfig); err != nil {
			logger.Error(err, "create config map failed")
			return nil, err
		}
	}
	logger.Info(fmt.Sprintf("created config map: %s", frpsConfig.Name))

	return frpsConfig, nil
}

func (r *EndpointReconciler) ensureEndpointPod(
	ctx context.Context,
	logger logr.Logger,
	endpoint *frpv1.Endpoint,
	frpsConfig *corev1.ConfigMap,
) (*corev1.Pod, error) {
	return nil, nil
}

func (r *EndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(
		&corev1.ConfigMap{}, endpointOwnerKey,
		func(rawObj runtime.Object) []string {
			config := rawObj.(*corev1.ConfigMap)
			owner := metav1.GetControllerOf(config)
			if owner == nil {
				return nil
			}
			if owner.APIVersion != apiGVStr || owner.Kind != KindEndpoint {
				return nil
			}
			return []string{owner.Name}
		},
	)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&frpv1.Endpoint{}).
		Complete(r)
}
