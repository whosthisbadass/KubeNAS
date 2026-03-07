# ─────────────────────────────────────────
# Stage 1: Build
# ─────────────────────────────────────────
FROM registry.access.redhat.com/ubi9/go-toolset:1.22 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o manager main.go

# ─────────────────────────────────────────
# Stage 2: Runtime (non-root UBI micro)
# ─────────────────────────────────────────
FROM registry.access.redhat.com/ubi9/ubi-micro:latest

# Add non-root user matching OpenShift restricted SCC.
RUN echo "kubenas:x:65532:65532:KubeNAS Operator:/:" >> /etc/passwd && \
    echo "kubenas:x:65532:" >> /etc/group

COPY --from=builder /workspace/manager /manager

USER 65532:65532
WORKDIR /

EXPOSE 8080 8081

ENTRYPOINT ["/manager"]
