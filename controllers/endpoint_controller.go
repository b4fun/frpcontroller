package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/b4fun/frpcontroller/pkg/frpconfig"

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
	frpcConfig, err := r.ensureEndpointConfigMap(ctx, logger, endpoint)
	if err != nil {
		return ctrl.Result{}, nil
	}

	frpcPod, err := r.ensureEndpointPod(ctx, logger, endpoint, frpcConfig)
	if err != nil {
		return ctrl.Result{}, nil
	}

	if frpcPod.Status.Phase == corev1.PodRunning {
		endpoint.Status = frpv1.EndpointStatus{
			State: frpv1.EndpointConnected,
		}
		if err := r.Status().Update(ctx, endpoint); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		endpoint.Status = frpv1.EndpointStatus{
			State: frpv1.EndpointDisconnected,
		}
		if err := r.Status().Update(ctx, endpoint); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{
		// update 10s later
		// TODO: can we trigger update in service side?
		RequeueAfter: time.Duration(10 * time.Second),
	}, nil
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
		frpcConfigList    corev1.ConfigMapList
		frpcConfig        *corev1.ConfigMap
		frpcConfigExisted bool
	)
	err := r.List(
		ctx, &frpcConfigList,
		client.InNamespace(endpoint.Namespace),
		client.MatchingFields{endpointOwnerKey: endpoint.Name},
	)
	if err != nil {
		logger.Error(err, "list endpoint config maps failed")
		return nil, err
	}
	if len(frpcConfigList.Items) == 0 {
		logger.Info("no endpoint config map found, will create")
		frpcConfigExisted = false
		frpcConfig = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Labels:       map[string]string{},
				Annotations:  map[string]string{},
				GenerateName: fmt.Sprintf("%s-frpc-", endpoint.Name),
				Namespace:    endpoint.Namespace,
			},
			Data: map[string]string{},
		}
		err := ctrl.SetControllerReference(endpoint, frpcConfig, r.Scheme)
		if err != nil {
			logger.Error(err, "set controller reference failed")
			return nil, err
		}
	} else {
		frpcConfigExisted = true
		frpcConfig = &frpcConfigList.Items[0]
		logger.Info(fmt.Sprintf(
			"found %d config maps, using %s",
			len(frpcConfigList.Items),
			frpcConfig.Name),
		)
	}

	var serviceList frpv1.ServiceList
	err = r.List(
		ctx, &serviceList,
		client.InNamespace(endpoint.Namespace),
		client.MatchingLabels{
			labelKeyEndpointName: endpoint.Name,
		},
	)
	if err != nil {
		logger.Error(err, "list services failed")
		return nil, err
	}

	frpcConfigContent, err := r.generateFrpcConfig(ctx, endpoint, &serviceList)
	if err != nil {
		logger.Error(err, "generate frpc config failed")
		return nil, err
	}

	if frpcConfig.Data == nil {
		frpcConfig.Data = map[string]string{}
	}
	frpcConfig.Data[frpcFileName] = frpcConfigContent

	if frpcConfigExisted {
		if err := r.Update(ctx, frpcConfig); err != nil {
			logger.Error(err, "update config map failed")
			return nil, err
		}
		logger.Info(fmt.Sprintf("updated config map: %s (%s)",
			frpcConfig.Name,
			frpcConfig.ResourceVersion,
		))
	} else {
		if err := r.Create(ctx, frpcConfig); err != nil {
			logger.Error(err, "create config map failed")
			return nil, err
		}
		logger.Info(fmt.Sprintf("created config map: %s (%s)",
			frpcConfig.Name,
			frpcConfig.ResourceVersion,
		))
	}

	return frpcConfig, nil
}

func (r *EndpointReconciler) generateFrpcConfig(
	ctx context.Context,
	endpoint *frpv1.Endpoint,
	services *frpv1.ServiceList,
) (string, error) {
	config := &frpconfig.FrpcConfig{
		Common: &frpconfig.ConfigCommon{
			ServerAddr: endpoint.Spec.Addr,
			ServerPort: int(endpoint.Spec.Port),
			Token:      endpoint.Spec.Token,
		},
		Apps: map[string]*frpconfig.ConfigApp{},
	}

	for _, service := range services.Items {
		if service.Annotations == nil {
			continue
		}
		localAddr, exists := service.Annotations[annotationKeyServiceClusterIP]
		if !exists {
			continue
		}

		for _, port := range service.Spec.Ports {
			appName := fmt.Sprintf("%s_%s", service.Name, port.Name)
			config.Apps[appName] = &frpconfig.ConfigApp{
				Type:       strings.ToLower(string(port.Protocol)),
				RemotePort: int(port.RemotePort),
				// NOTE: the service is exposed with remote port
				LocalPort: int(port.RemotePort),
				LocalAddr: localAddr,
			}
		}
	}

	return config.GenerateIni()
}

func (r *EndpointReconciler) ensureEndpointPod(
	ctx context.Context,
	logger logr.Logger,
	endpoint *frpv1.Endpoint,
	frpcConfig *corev1.ConfigMap,
) (*corev1.Pod, error) {
	var (
		podList corev1.PodList
		pod     *corev1.Pod
	)
	err := r.List(
		ctx, &podList,
		client.InNamespace(endpoint.Namespace),
		client.MatchingFields{endpointOwnerKey: endpoint.Name},
	)
	if err != nil {
		logger.Error(err, "list endpoint pods failed")
		return nil, err
	}

	for _, p := range podList.Items {
		if len(p.Annotations) > 0 {
			configVersion, exists := p.Annotations[annotationKeyEndpointPodConfigVersion]
			if exists && configVersion == frpcConfig.ResourceVersion {
				pod = &p
				logger.Info(fmt.Sprintf("found pod with updated config: %s", p.Name))
				break
			}
		}
	}
	for _, p := range podList.Items {
		if pod != nil && p.Name == pod.Name {
			// retain
			continue
		}
		err = r.Delete(ctx, &p)
		if err != nil {
			logger.Error(err, fmt.Sprintf("delete pod %s failed", p.Name))
			return nil, err
		}
		logger.Info(fmt.Sprintf("deleted pod %s", p.Name))
	}

	if pod != nil {
		return pod, nil
	}

	const (
		frpcVolumeName    = "frpc-config"
		frpcContainerName = "frpc"
	)

	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{},
			Annotations: map[string]string{
				annotationKeyEndpointPodConfigVersion: frpcConfig.ResourceVersion,
			},
			GenerateName: fmt.Sprintf("%s-frpc-", endpoint.Name),
			Namespace:    endpoint.Namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: frpcVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: frpcConfig.Name,
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    frpcContainerName,
					Image:   frpDockerImage,
					Command: []string{"/opt/frp/frpc"},
					Args:    []string{"-c", "/data/frpc.ini"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      frpcVolumeName,
							ReadOnly:  true,
							MountPath: "/data/frpc.ini",
							SubPath:   "frpc.ini",
						},
					},
				},
			},
		},
	}
	err = ctrl.SetControllerReference(endpoint, pod, r.Scheme)
	if err != nil {
		logger.Error(err, "set controller reference failed")
		return nil, err
	}
	err = r.Create(ctx, pod)
	if err != nil {
		logger.Error(err, fmt.Sprintf("create pod %s failed", pod.Name))
		return nil, err
	}
	logger.Info(fmt.Sprintf("created pod: %s", pod.Name))

	return pod, nil
}

func (r *EndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	err = mgr.GetFieldIndexer().IndexField(
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
	err = mgr.GetFieldIndexer().IndexField(
		&corev1.Pod{}, endpointOwnerKey,
		func(rawObj runtime.Object) []string {
			pod := rawObj.(*corev1.Pod)
			owner := metav1.GetControllerOf(pod)
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
