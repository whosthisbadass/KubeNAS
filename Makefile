# Image URL to use all building/pushing image targets
IMG ?= controller:latest
BUNDLE_IMG ?= controller-bundle:latest

# Produce CRDs that work back to Kubernetes 1.27
CRD_OPTIONS ?= "crd:trivialVersions=true"

KUSTOMIZE ?= $(shell which kustomize)
CONTROLLER_GEN ?= $(shell which controller-gen)
ENVTEST ?= $(shell which setup-envtest)

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: all
all: build

##@ Development

.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use 1.29 -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: manifests generate fmt vet
	go build -o bin/manager ./cmd/main.go

.PHONY: run
run: manifests generate fmt vet
	go run ./cmd/main.go

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

##@ Deployment

.PHONY: install
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=true -f -

.PHONY: deploy
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=true -f -

##@ Bundle

.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q || true
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version 0.0.1

.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push:
	docker push $(BUNDLE_IMG)

##@ Build Dependencies

.PHONY: kustomize
kustomize: $(LOCALBIN)
	@if [ -z "$(KUSTOMIZE)" ]; then GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@latest; fi
	$(eval KUSTOMIZE=$(LOCALBIN)/kustomize)

.PHONY: controller-gen
controller-gen: $(LOCALBIN)
	@if [ -z "$(CONTROLLER_GEN)" ]; then GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.5; fi
	$(eval CONTROLLER_GEN=$(LOCALBIN)/controller-gen)

.PHONY: envtest
envtest: $(LOCALBIN)
	@if [ -z "$(ENVTEST)" ]; then GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest; fi
	$(eval ENVTEST=$(LOCALBIN)/setup-envtest)
