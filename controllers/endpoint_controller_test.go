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
		resourcePollingInterval = "10s"
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

	g.It("should create endpoint", func() {
		ctx := context.Background()

		endpointSpec := frpv1.EndpointSpec{
			Addr:  frpsDeploy.Endpoint,
			Port:  frpsDeploy.Port,
			Token: frpsDeploy.Token,
		}
		endpointToCreate := &frpv1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "test-endpoint",
			},
			Spec: endpointSpec,
		}
		endpointName := client.ObjectKey{
			Namespace: endpointToCreate.Namespace,
			Name:      endpointToCreate.Name,
		}

		g.By("creating an endpoint object in the cluster")
		err := k8sClient.Create(ctx, endpointToCreate)
		m.Expect(err).NotTo(
			m.HaveOccurred(),
			"create an endpoint",
		)

		g.By("getting the endpoint object")
		endpointCreated := &frpv1.Endpoint{}
		m.Eventually(func() error {
			err := k8sClient.Get(ctx, endpointName, endpointCreated)
			if err != nil {
				return err
			}

			endpointStatusString := fmt.Sprintf("endpoint status: %+v", endpointCreated.Status)
			log.Log.Info(endpointStatusString)

			if endpointCreated.Status.State != frpv1.EndpointConnected {
				return errors.New(endpointStatusString)
			}

			return nil
		}, resourcePollingTimeout, resourcePollingInterval).
			ShouldNot(m.HaveOccurred())

		g.By("validating created endpoint properties")
		m.Expect(endpointCreated.Namespace).To(m.Equal(testNamespace))
		m.Expect(endpointCreated.Name).To(m.Equal(endpointName.Name))

		var (
			configMapCreated *corev1.ConfigMap
			podCreated       *corev1.Pod
		)

		g.By("inspecting created config map", func() {
			m.Eventually(func() error {
				var (
					configMapList corev1.ConfigMapList
					err           error
				)
				err = k8sClient.List(
					ctx, &configMapList,
					client.InNamespace(testNamespace),
				)
				if err != nil {
					return err
				}
				for _, configMap := range configMapList.Items {
					for _, owner := range configMap.OwnerReferences {
						if owner.Name == endpointCreated.Name {
							configMapCreated = &configMap
							return nil
						}
					}
				}
				return errors.New("endpoint config map not found")
			}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

			g.By("inspecting endpoint properties")
			m.Expect(configMapCreated.Namespace).To(m.Equal(testNamespace))
			m.Expect(configMapCreated.Data).NotTo(m.BeEmpty())
		})

		g.By("inspecting created pod", func() {
			m.Eventually(func() error {
				var (
					podList corev1.PodList
					err     error
				)
				err = k8sClient.List(
					ctx, &podList,
					client.InNamespace(testNamespace),
				)
				if err != nil {
					return err
				}
				for _, pod := range podList.Items {
					for _, owner := range pod.OwnerReferences {
						if owner.Name == endpointCreated.Name {
							podCreated = &pod
							return nil
						}
					}
				}
				return errors.New("endpoint pod not found")
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
})
