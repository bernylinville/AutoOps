# Security Policy

We take security seriously at AutoOps. We appreciate the efforts of security researchers who help keep our platform safe.

## Reporting a Vulnerability

**Do not open a public GitHub issue** for security vulnerabilities — this could expose the vulnerability before a fix is available.

Instead, report it through one of these private channels:

- **Email:** [security@autoops.dev](mailto:security@autoops.dev)
- **GitHub Security Advisory:** [Open a private advisory](https://github.com/bernylinville/AutoOps/security/advisories/new)

We will acknowledge your report within 48 hours and work with you to understand and resolve the issue as quickly as possible.

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | Best-effort, upgrade recommended |

## Scope

The following are in scope for our security program:

- The AutoOps API server (`api/`)
- The AutoOps web frontend (`web/`)
- Deployment configurations (`docker/`, `deploy/`, `charts/`)
- CI/CD pipeline configurations (`.github/workflows/`)

Third-party services (PostgreSQL, Valkey, Prometheus, etc.) should be reported to their respective maintainers.
