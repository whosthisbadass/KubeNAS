# KubeNAS Operator Makefile
# Provides standard development, build, and deployment targets.

# ── Configuration ──────────────────────────────────────────────────
REGISTRY     ?= ghcr.io/kubenas
VERSION      ?= 0.1.0
IMG_OPERATOR ?= $(REGISTRY)/operator:$(VERSION)
IMG_AGENT    ?= $(REGISTRY)/node-agent:$(VERSION)

# Tools
ENVTEST_K8S_VERSION ?= 1.29
CONTROLLER_GEN      ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@latest
KUSTOMIZE           ?= go run sigs.k8s.io/kustomize/kustomize/v5@latest
GOLANGCI_LINT       ?= go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest
ENVTEST             ?= go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# ── Go Targets ─────────────────────────────────────────────────────

.PHONY: all
all: fmt vet build

.PHONY: fmt
fmt:
	cd operator && go fmt ./...
	cd node-agent && go fmt ./...

.PHONY: vet
vet:
	cd operator && go vet ./...
	cd node-agent && go vet ./...

.PHONY: lint
lint:
	cd operator && $(GOLANGCI_LINT) run ./...
	cd node-agent && $(GOLANGCI_LINT) run ./...

.PHONY: build
build: build-operator build-agent

.PHONY: build-operator
build-operator:
	cd operator && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags="-X main.version=$(VERSION)" -o bin/manager ./main.go

.PHONY: build-agent
build-agent:
	cd node-agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags="-X main.version=$(VERSION)" -o bin/kubenas-node-agent ./cmd/kubenas-node-agent/main.go

# ── Test Targets ───────────────────────────────────────────────────

.PHONY: test
test:
	cd operator && KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit:
	cd operator && go test ./... -run TestUnit

.PHONY: test-integration
test-integration:
	cd operator && KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test ./... -run TestIntegration -v

# ── Code Generation ────────────────────────────────────────────────

.PHONY: generate
generate:
	cd operator && $(CONTROLLER_GEN) object paths="./api/..."

.PHONY: manifests
manifests:
	cd operator && $(CONTROLLER_GEN) crd paths="./api/..." output:crd:artifacts:config=../deploy/crds
	cd operator && $(CONTROLLER_GEN) rbac:roleName=kubenas-operator paths="./controllers/..." output:rbac:artifacts:config=../deploy/rbac

# ── Docker Targets ─────────────────────────────────────────────────

.PHONY: docker-build
docker-build: docker-build-operator docker-build-agent

.PHONY: docker-build-operator
docker-build-operator:
	docker build -t $(IMG_OPERATOR) -f operator/Dockerfile operator/

.PHONY: docker-build-agent
docker-build-agent:
	docker build -t $(IMG_AGENT) -f node-agent/Dockerfile node-agent/

.PHONY: docker-push
docker-push:
	docker push $(IMG_OPERATOR)
	docker push $(IMG_AGENT)

# ── Deploy Targets ─────────────────────────────────────────────────

.PHONY: install
install:
	bash scripts/install.sh

.PHONY: deploy-crds
deploy-crds:
	kubectl apply -f deploy/crds/

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/examples/node-agent-daemonset.yaml --ignore-not-found
	kubectl delete -f deploy/operator.yaml --ignore-not-found
	kubectl delete -f deploy/rbac/rbac.yaml --ignore-not-found
	kubectl delete -f deploy/crds/ --ignore-not-found
	kubectl delete -f deploy/scc/scc.yaml --ignore-not-found 2>/dev/null || true

# ── Tools ──────────────────────────────────────────────────────────

.PHONY: tools
tools:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	go install sigs.k8s.io/kustomize/kustomize/v5@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: help
help:
	@echo "KubeNAS Operator Makefile"
	@echo ""
	@echo "Build targets:"
	@echo "  make build          - Build operator and agent binaries"
	@echo "  make docker-build   - Build container images"
	@echo "  make docker-push    - Push images to registry"
	@echo ""
	@echo "Test targets:"
	@echo "  make test           - Run all tests with envtest"
	@echo "  make test-unit      - Run unit tests only"
	@echo ""
	@echo "Code gen targets:"
	@echo "  make generate       - Regenerate deepcopy functions"
	@echo "  make manifests      - Regenerate CRD and RBAC manifests"
	@echo ""
	@echo "Deploy targets:"
	@echo "  make install        - Full SNO install via install.sh"
	@echo "  make deploy-crds    - Apply CRDs only"
	@echo "  make undeploy       - Remove all KubeNAS resources"
