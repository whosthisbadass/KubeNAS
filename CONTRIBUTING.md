# Contributing to KubeNAS

Thanks for helping improve KubeNAS.

## Development Workflow

1. Fork the repository and create a feature branch.
2. Keep changes scoped to one concern (CRD, controller, docs, node-agent, etc.).
3. Add or update tests for all behavior changes.
4. Run local checks before opening a PR.
5. Open a PR using the project template and complete the checklist.

## Local Setup

```bash
git clone https://github.com/<your-user>/kubenas.git
cd kubenas
make tools
make test
```

For operator-focused development, use `operator-sdk` and `envtest`.

## Coding Standards

### Go

- follow `gofmt` and `goimports`
- prefer small, composable reconciler methods
- avoid hidden side effects inside reconcile loops
- return explicit Kubernetes conditions for state transitions

### API/CRD Design

- all fields must include clear godoc comments
- use enums and validation for constrained values
- avoid breaking API changes within a minor series
- model status conditions with `type`, `status`, `reason`, `message`

### Node Agent

- host actions must be idempotent and retry-safe
- command execution must be audited/logged
- never assume fixed device names across reboots

## Testing Strategy

Contributors should test at multiple layers:

1. **Unit tests**
   - scheduler scoring
   - CRD validation helpers
   - config rendering functions
2. **Controller tests**
   - reconcile behavior with `envtest`
   - condition transitions and error paths
3. **Integration tests**
   - deploy CRDs/controllers into local cluster
   - verify end-to-end resource state

Suggested commands:

```bash
make fmt
make lint
make test
make test-integration
```

## Pull Request Process

Each PR should include:

- concise summary and motivation
- linked issue (`Fixes #123` when applicable)
- API impact notes (if CRDs changed)
- upgrade/migration notes for breaking behavior
- test evidence (logs, screenshots, or command output)

PRs require at least one maintainer review and passing CI.

## Commit Message Guidance

Use clear, action-oriented messages.

Examples:

- `feat(controller): add parity schedule cronjob reconciliation`
- `fix(agent): handle missing smartctl binary gracefully`
- `docs(crds): add CachePool and RebalanceJob examples`

## Security Reporting

Do not open public issues for vulnerabilities. Follow [SECURITY.md](./SECURITY.md).
