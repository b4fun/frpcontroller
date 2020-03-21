
# Release version
RELEASE ?= latest
# Image URL to use all building/pushing image targets
IMG ?= b4fun/frpcontroller:${RELEASE}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests in local
test-local: generate fmt vet manifests
	USE_EXISTING_CLUSTER=true \
	go test ./... -v -ginkgo.v -ginkgo.progress -coverprofile cover.out

# Run tests in ci
# TODO(hbc): setup kind in CI
test-ci: generate fmt vet manifests setup-ci
	USE_EXISTING_CLUSTER=true \
	go test ./... -v -ginkgo.v -ginkgo.progress -coverprofile cover.out
	go mod vendor

# Setup in ci
setup-ci:
	go mod vendor

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
# ...with cn optimization 😂
docker-build:
	docker build . -t ${IMG} \
		--build-arg=goproxy="https://goproxy.cn" \
		--build-arg=nonroot_image="gcr.azk8s.cn/distroless/static:nonroot"

# Build the docker image in ci envrionment
docker-build-ci:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# make an release
release: release-prepare release-kustomize release-kustomize-cn

release-prepare:
	@echo "Building for ${RELEASE} (${IMG})"
	mkdir -p release/${RELEASE}

release-kustomize:
	mkdir -p config/rel-${RELEASE}
	cp config/default/*.yaml config/rel-${RELEASE}/
	cd config/rel-${RELEASE} && kustomize edit set image controller=${IMG}
	cd config/rel-${RELEASE} && kustomize edit add label frp.go.build4.fun/release:${RELEASE}
	kustomize build config/rel-${RELEASE} > release/${RELEASE}/install.yaml
	rm -r config/rel-${RELEASE}

release-kustomize-cn:
	mkdir -p config/rel-${RELEASE}
	cp config/default/*.yaml config/rel-${RELEASE}/
	cd config/rel-${RELEASE} && kustomize edit set image controller=dockerhub.azk8s.cn/${IMG}
	cd config/rel-${RELEASE} && kustomize edit set image gcr.io/kubebuilder/kube-rbac-proxy=gcr.azk8s.cn/kubebuilder/kube-rbac-proxy
	cd config/rel-${RELEASE} && kustomize edit add label frp.go.build4.fun/release:${RELEASE}
	kustomize build config/rel-${RELEASE} > release/${RELEASE}/install-cn.yaml
	rm -r config/rel-${RELEASE}
