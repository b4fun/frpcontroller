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

var _ = g.Describe("ServiceController", func() {
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

	getServiceService := func(namespace string, serviceName string) *corev1.Service {
		serviceRetrieved := &corev1.Service{}
		m.Eventually(func() error {
			var (
				serviceList corev1.ServiceList
				err         error
			)
			ctx := context.Background()
			err = k8sClient.List(
				ctx, &serviceList,
				client.InNamespace(namespace),
			)
			if err != nil {
				return err
			}
			foundAlready := false
			for _, service := range serviceList.Items {
				isOwnByService := false
				for _, owner := range service.OwnerReferences {
					if owner.Name == serviceName {
						isOwnByService = true
						break
					}
				}
				if !isOwnByService {
					continue
				}
				if isOwnByService && foundAlready {
					return fmt.Errorf(
						"found multiple pods owned by the endpoint: %s %s",
						serviceRetrieved.Name, service.Name,
					)
				}
				foundAlready = true
				*serviceRetrieved = service
			}
			if foundAlready {
				return nil
			}
			return errors.New("corev1.service not found")
		}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

		return serviceRetrieved
	}

	g.It("should create service without endpoint", func() {
		ctx := context.Background()

		serviceSpec := frpv1.ServiceSpec{
			Endpoint: "test-endpoint",
			Ports: []frpv1.ServicePort{
				{
					Name:       "test-port",
					Protocol:   frpv1.ServicePortTCP,
					LocalPort:  3333,
					RemotePort: 3333,
				},
			},
			Selector: map[string]string{
				"foo": "bar",
			},
			ServiceLabels: map[string]string{
				"labelFoo": "bar",
			},
		}
		serviceToCreate := &frpv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    testNamespace,
				GenerateName: "frpc-service-",
			},
			Spec: serviceSpec,
		}

		err := k8sClient.Create(ctx, serviceToCreate)
		m.Expect(err).NotTo(m.HaveOccurred(), "create service")

		serviceName := client.ObjectKey{
			Namespace: serviceToCreate.Namespace,
			Name:      serviceToCreate.Name,
		}
		serviceCreated := &frpv1.Service{}
		m.Eventually(func() error {
			var (
				service frpv1.Service
				err     error
			)
			err = k8sClient.Get(ctx, serviceName, &service)
			if err != nil {
				return err
			}

			if service.Status.State == "" {
				return errors.New("service has no status yet")
			}

			*serviceCreated = service
			return nil
		}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

		m.Expect(serviceCreated.Status.State).To(m.Equal(frpv1.ServiceStateInactive))
		corev1Service := getServiceService(serviceCreated.Namespace, serviceCreated.Name)
		for k, v := range serviceSpec.Selector {
			m.Expect(corev1Service.Spec.Selector).To(m.HaveKeyWithValue(k, v))
		}
		for k, v := range serviceSpec.ServiceLabels {
			m.Expect(corev1Service.Labels).To(m.HaveKeyWithValue(k, v))
		}
	})

	g.It("should create service with endpoint", func() {
		ctx := context.Background()

		endpoint, err := createEndpoint(ctx, k8sClient, testNamespace, frpsDeploy)
		m.Expect(err).NotTo(m.HaveOccurred())
		log.Log.Info(fmt.Sprintf("created endpoint: %s", endpoint.Name))

		serviceSpec := frpv1.ServiceSpec{
			Endpoint: endpoint.Name,
			Ports: []frpv1.ServicePort{
				{
					Name:       "test-port",
					Protocol:   frpv1.ServicePortTCP,
					LocalPort:  3333,
					RemotePort: 3333,
				},
			},
			Selector: map[string]string{
				"foo": "bar",
			},
			ServiceLabels: map[string]string{
				"labelFoo": "bar",
			},
		}

		serviceToCreate := &frpv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    testNamespace,
				GenerateName: "frpc-service-",
			},
			Spec: serviceSpec,
		}

		err = k8sClient.Create(ctx, serviceToCreate)
		m.Expect(err).NotTo(m.HaveOccurred(), "create service")

		serviceName := client.ObjectKey{
			Namespace: serviceToCreate.Namespace,
			Name:      serviceToCreate.Name,
		}
		serviceCreated := &frpv1.Service{}
		m.Eventually(func() error {
			var (
				service frpv1.Service
				err     error
			)
			err = k8sClient.Get(ctx, serviceName, &service)
			if err != nil {
				return err
			}

			if service.Status.State != frpv1.ServiceStateActive {
				return fmt.Errorf("service is not active yet: %s", service.Status.State)
			}

			*serviceCreated = service
			return nil
		}, resourcePollingTimeout, resourcePollingInterval).ShouldNot(m.HaveOccurred())

		m.Expect(serviceCreated.Status.State).To(m.Equal(frpv1.ServiceStateActive))
		m.Expect(serviceCreated.Annotations).To(m.HaveKey(annotationKeyServiceClusterIP))
		corev1Service := getServiceService(serviceCreated.Namespace, serviceCreated.Name)
		for k, v := range serviceSpec.Selector {
			m.Expect(corev1Service.Spec.Selector).To(m.HaveKeyWithValue(k, v))
		}
		for k, v := range serviceSpec.ServiceLabels {
			m.Expect(corev1Service.Labels).To(m.HaveKeyWithValue(k, v))
		}
	})

})
