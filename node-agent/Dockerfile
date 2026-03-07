# ─────────────────────────────────────────
# Stage 1: Build
# ─────────────────────────────────────────
FROM registry.access.redhat.com/ubi9/go-toolset:1.22 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o kubenas-node-agent ./cmd/kubenas-node-agent/main.go

# ─────────────────────────────────────────
# Stage 2: Runtime (UBI minimal for tool access)
# The node agent runs privileged and needs access to host binaries
# via hostPath mounts (/host/bin). The image itself is minimal.
# ─────────────────────────────────────────
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN microdnf install -y util-linux procps-ng && \
    microdnf clean all

COPY --from=builder /workspace/kubenas-node-agent /kubenas-node-agent

# Agent runs as root for disk management.
USER 0

WORKDIR /

EXPOSE 9100

ENTRYPOINT ["/kubenas-node-agent"]
