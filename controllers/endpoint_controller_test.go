package controllers

import (
	"context"
	"errors"
	"fmt"

	g "github.com/onsi/ginkgo"
	m "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	frpv1 "github.com/b4fun/frpcontroller/api/v1"
)

var _ = g.Describe("EndpointController", func() {
	const (
		resourcePollingTimeout  = "1m"
		resourcePollingInterval = "2s"
	)

	var (
		frpsDeploy    *frpsDeployStatus
		testNamespace string
	)

	g.BeforeEach(func(done g.Done) {
		ctx := context.Background()
		var err error

		g.By("create test namespace")
		testNamespace, err = createNamespace(ctx, k8sClient, "frp-test-")
		m.Expect(err).NotTo(m.HaveOccurred(), "create namespace")
		log.Log.Info(fmt.Sprintf("created namespace: %s", testNamespace))

		g.By("deploying frps server")
		frps := frpsSettings{
			Port:  3333,
			Token: "supersecret",
		}
		frpsDeploy, err = frps.DeployToCluster(ctx, k8sClient, testNamespace)
		m.Expect(err).NotTo(m.HaveOccurred(), "deploy frps")
		log.Log.Info(fmt.Sprintf("deployed frps: %s", frpsDeploy))

		close(done)
	}, 300)

	g.AfterEach(func(done g.Done) {
		err := deleteNamespace(context.Background(), k8sClient, testNamespace)
		m.Expect(err).NotTo(m.HaveOccurred(), "delete namespace")

		close(done)
	}, 300)

	waitEndpointReady := func(namespace string, name string) *frpv1.Endpoint {
		g.By("waiting the endpoint to be ready")
		ctx := context.Background()
		endpointName := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		endpoint := &frpv1.Endpoint{}
		m.Eventually(func() error {
			err := k8sClient.Get(ctx, endpointName, endpoint)
			if err != nil {
				return err
			}

			endpointStatusString := fmt.Sprintf("endpoint status: %+v", endpoint.Status)
			log.Log.Info(endpointStatusString)

			if endpoint.Status.State != frpv1.EndpointConnected {
				return errors.New(endpointStatusString)
			}

			return nil
		}, resourcePollingTimeout, resourcePollingInterval).
			ShouldNot(m.HaveOccurred())

		return endpoint
	}

	createEndpoint := func(namespace string, name string) *frpv1.Endpoint {
		ctx := context.Background()

		endpointSpec := frpv1.EndpointSpec{
			Addr:  frpsDeploy.Endpoint,
			Port:  frpsDeploy.Port,
			Token: frpsDeploy.Token,
		}
		endpointToCreate := &frpv1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: endpointSpec,
		}

		g.By("creating an endpoint object in the cluster")
		err := k8sClient.Create(ctx, endpointToCreate)
		m.Expect(err).NotTo(
			m.HaveOccurred(),
			"create an endpoint",
		)

		return waitEndpointReady(endpointToCreate.Namespace, endpointToCreate.Name)
	}

	getEndpointConfigMap := func(namespace string, name string) *corev1.ConfigMap {
		var configMapRetrieved *corev1.ConfigMap
		m.Eventually(func() error {
			var (
				configMapList corev1.ConfigMapList
				err           error
			)
			ctx := context.Background()
			err = k8sClient.List(
				ctx, &configMapList,
				client.InNamespace(namespace),
			)
			if err != nil {
				return err
			}
			foundAlready := false
			for _, configMap := range configMapList.Items {
				isOwnByEndpoint := false
				for _, owner := range configMap.OwnerReferences {
					if owner.Name == name {
						isOwnByEndpoint = true
						break
					}
				}
				if !isOwnByEndpoint {
					continue
				}
				if isOwnByEndpoint && foundAlready {
					return fmt.Errorf(
						"found multiple config maps owned by the endpoint: %s %s",
						configMapRetrieved.Name, configMap.Name,
					)
				}
				foundAlready = true
				configMapRetrieved = &configMap
			}
			if foundAlready {
				return nil
			}
			return errors.New("endpoint config map not found")
		}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

		return configMapRetrieved
	}

	getEndpointPod := func(namespace string, name string) *corev1.Pod {
		var podRetrieved *corev1.Pod
		m.Eventually(func() error {
			var (
				podList corev1.PodList
				err     error
			)
			ctx := context.Background()
			err = k8sClient.List(
				ctx, &podList,
				client.InNamespace(namespace),
			)
			if err != nil {
				return err
			}
			foundAlready := false
			for _, pod := range podList.Items {
				isOwnByEndpoint := false
				if pod.Status.Phase != corev1.PodRunning {
					// skip non-running pods
					continue
				}
				for _, owner := range pod.OwnerReferences {
					if owner.Name == name {
						isOwnByEndpoint = true
						break
					}
				}
				if !isOwnByEndpoint {
					continue
				}
				if isOwnByEndpoint && foundAlready {
					return fmt.Errorf(
						"found multiple pods owned by the endpoint: %s %s",
						podRetrieved.Name, pod.Name,
					)
				}
				foundAlready = true
				podRetrieved = &pod
			}
			if foundAlready {
				return nil
			}
			return errors.New("endpoint pod not found")
		}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

		return podRetrieved
	}

	g.It("should create endpoint", func() {
		endpointName := "test-endpoint"
		endpointCreated := createEndpoint(testNamespace, endpointName)

		g.By("validating created endpoint properties")
		m.Expect(endpointCreated.Namespace).To(m.Equal(testNamespace))
		m.Expect(endpointCreated.Name).To(m.Equal(endpointName))

		var (
			configMapCreated *corev1.ConfigMap
			podCreated       *corev1.Pod
		)

		g.By("inspecting created config map", func() {
			g.By("getting created config map")
			configMapCreated = getEndpointConfigMap(endpointCreated.Namespace, endpointCreated.Name)

			g.By("inspecting config map properties")
			m.Expect(configMapCreated.Namespace).To(m.Equal(testNamespace))
			m.Expect(configMapCreated.Data).NotTo(m.BeEmpty())
		})

		g.By("inspecting created pod", func() {
			g.By("getting created pod")
			podCreated = getEndpointPod(endpointCreated.Namespace, endpointCreated.Name)

			g.By("inspecting pod properties")
			m.Expect(podCreated.Namespace).To(m.Equal(testNamespace))
			m.Expect(podCreated.Status.Phase).To(m.Equal(corev1.PodRunning))
			m.Expect(podCreated.Annotations).To(
				m.HaveKeyWithValue(
					annotationKeyEndpointPodConfigVersion,
					configMapCreated.ResourceVersion,
				),
				"pod should use latest endpoint",
			)
		})
	})

	g.It("should update endpoint", func() {
		ctx := context.Background()

		newFrpsSettings := frpsSettings{
			Port:  3333,
			Token: "foobar",
		}
		newFrps, err := newFrpsSettings.DeployToCluster(ctx, k8sClient, testNamespace)
		m.Expect(err).NotTo(m.HaveOccurred())

		endpointName := "test-endpoint"
		endpointCreated := createEndpoint(testNamespace, endpointName)

		g.By("updating the endpoint settings")
		endpointCreated.Spec.Addr = newFrps.Endpoint
		endpointCreated.Spec.Port = int32(newFrps.Port)
		endpointCreated.Spec.Token = newFrps.Token
		err = k8sClient.Update(ctx, endpointCreated)
		m.Expect(err).NotTo(m.HaveOccurred())

		endpointUpdated := waitEndpointReady(endpointCreated.Namespace, endpointCreated.Name)

		var (
			configMapCreated *corev1.ConfigMap
			podCreated       *corev1.Pod
		)

		g.By("inspecting created config map", func() {
			g.By("getting created config map")
			configMapCreated = getEndpointConfigMap(endpointUpdated.Namespace, endpointUpdated.Name)

			g.By("inspecting config map settings")
			m.Expect(configMapCreated.Data).NotTo(m.BeEmpty())
			m.Expect(configMapCreated.Data).To(m.HaveKey(frpcFileName))
			frpcFileContent := configMapCreated.Data[frpcFileName]
			m.Expect(frpcFileContent).To(m.ContainSubstring(newFrps.Token))
		})

		g.By("inspecting created pod", func() {
			g.By("getting created pod")
			m.Eventually(func() error {
				pod := getEndpointPod(endpointCreated.Namespace, endpointCreated.Name)

				if v, ok := pod.Annotations[annotationKeyEndpointPodConfigVersion]; ok {
					if v == configMapCreated.ResourceVersion {
						podCreated = pod
						return nil
					}
				}
				return errors.New("endpoint pod with latest config map is not ready")

			}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

			g.By("inspecting pod properties")
			m.Expect(podCreated.Namespace).To(m.Equal(testNamespace))
			m.Expect(podCreated.Status.Phase).To(m.Equal(corev1.PodRunning))
			m.Expect(podCreated.Annotations).To(
				m.HaveKeyWithValue(
					annotationKeyEndpointPodConfigVersion,
					configMapCreated.ResourceVersion,
				),
				"pod should use latest endpoint",
			)
		})
	})

	g.It("should delete endpoint", func() {
		ctx := context.Background()

		endpointName := "test-endpoint"
		endpointCreated := createEndpoint(testNamespace, endpointName)

		err := k8sClient.Delete(ctx, endpointCreated)
		m.Expect(err).NotTo(m.HaveOccurred(), "delete endpoint")
	})
})
