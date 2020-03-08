package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	frpv1 "github.com/b4fun/frpcontroller/api/v1"
)

type retryOption struct {
	RetryAttempts uint
	RetryPolling  time.Duration
}

func (r *retryOption) Retry(fn func() error) error {
	var lastErr error

	if err := fn(); err == nil {
		return nil
	} else {
		lastErr = err
	}

	for i := uint(1); i < r.RetryAttempts; i++ {
		time.Sleep(r.RetryPolling)
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return lastErr
}

func createNamespace(
	ctx context.Context,
	k8sClient client.Client,
	generateName string,
) (string, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{GenerateName: generateName},
	}
	err := k8sClient.Create(ctx, ns)
	if err != nil {
		return "", err
	}
	return ns.Name, nil
}

func deleteNamespace(
	ctx context.Context,
	k8sClient client.Client,
	name string,
) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err := k8sClient.Delete(ctx, ns)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

type frpsDeployStatus struct {
	Endpoint string
	Port     int32
	Token    string
}

func (s frpsDeployStatus) String() string {
	return fmt.Sprintf("endpoint=%s:%d token=%s", s.Endpoint, s.Port, s.Token)
}

type frpsSettings struct {
	Port  int32
	Token string
}

func (s *frpsSettings) buildFrpsConfig(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "frps-config-",
			Namespace:    namespace,
		},
		Data: map[string]string{
			"frps.ini": fmt.Sprintf(`
[common]
bind_port = %d
token = %s
log_level=info
`, s.Port, s.Token),
		},
	}
}

func (s *frpsSettings) buildFrpsPod(
	namespace string,
	configMap *corev1.ConfigMap,
) *corev1.Pod {
	const (
		configVolume = "config-volume"
	)

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "frps-pod-",
			Namespace:    namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "frps",
					Image:   frpDockerImage,
					Command: []string{"/opt/frp/frps"},
					Args:    []string{"-c", "/data/frps.ini"},
					Ports: []corev1.ContainerPort{
						{ContainerPort: s.Port},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolume,
							MountPath: "/data",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: configVolume,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMap.Name,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  "frps.ini",
									Path: "frps.ini",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *frpsSettings) DeployToCluster(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
) (*frpsDeployStatus, error) {
	configMap := s.buildFrpsConfig(namespace)
	err := k8sClient.Create(ctx, configMap)
	if err != nil {
		return nil, err
	}
	log.Log.Info(fmt.Sprintf("created config map: %s", configMap.Name))

	pod := s.buildFrpsPod(namespace, configMap)
	err = k8sClient.Create(ctx, pod)
	if err != nil {
		return nil, err
	}
	log.Log.Info(fmt.Sprintf("created pod: %s", pod.Name))

	podName := client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}

	deployStatus := &frpsDeployStatus{}
	retryOpt := retryOption{
		RetryAttempts: 30,
		RetryPolling:  time.Duration(10) * time.Second,
	}
	retryErr := retryOpt.Retry(func() error {
		var (
			podLatest corev1.Pod
			err       error
		)
		err = k8sClient.Get(ctx, podName, &podLatest)
		if err != nil {
			return err
		}
		log.Log.Info(fmt.Sprintf("pod status: %s", podLatest.Status.Phase))
		if podLatest.Status.Phase == corev1.PodRunning {
			deployStatus.Endpoint = podLatest.Status.PodIP
			deployStatus.Port = s.Port
			deployStatus.Token = s.Token
			return nil
		}

		log.Log.Info("pod is not ready, retry...")
		return errors.New("pod is not ready")
	})
	if retryErr != nil {
		return nil, retryErr
	}

	return deployStatus, nil
}

func waitEndpointReady(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	name string,
	retryOption *retryOption,
) (*frpv1.Endpoint, error) {
	endpointReady := &frpv1.Endpoint{}
	var retryErr error
	retryErr = retryOption.Retry(func() error {
		endpointName := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		endpoint := &frpv1.Endpoint{}
		if err := k8sClient.Get(ctx, endpointName, endpoint); err != nil {
			return err
		}

		endpointStatusString := fmt.Sprintf("endpoint status: %+v", endpoint.Status)
		log.Log.Info(endpointStatusString)
		if endpoint.Status.State != frpv1.EndpointConnected {
			return errors.New(endpointStatusString)
		}

		*endpointReady = *endpoint
		return nil
	})

	if retryErr != nil {
		return nil, retryErr
	}
	return endpointReady, nil
}

func createEndpoint(
	ctx context.Context,
	k8sClient client.Client,
	namespace string,
	frpsDeploy *frpsDeployStatus,
) (*frpv1.Endpoint, error) {
	endpointSpec := frpv1.EndpointSpec{
		Addr:  frpsDeploy.Endpoint,
		Port:  frpsDeploy.Port,
		Token: frpsDeploy.Token,
	}
	endpointToCreate := &frpv1.Endpoint{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "frpc-endpoint-",
		},
		Spec: endpointSpec,
	}

	err := k8sClient.Create(ctx, endpointToCreate)
	if err != nil {
		return nil, err
	}

	return waitEndpointReady(
		ctx,
		k8sClient,
		endpointToCreate.Namespace,
		endpointToCreate.Name,
		&retryOption{
			RetryAttempts: 120,
			RetryPolling:  time.Duration(1) * time.Second,
		},
	)
}
