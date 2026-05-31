# Security Policy

## Supported versions

`r53ctl` is pre-1.0. Until a stable release exists, security fixes target the latest release only.

## Reporting a vulnerability

Please do not open a public GitHub issue for security reports.

Report suspected vulnerabilities by emailing:

```text
kevin.espineira@gmail.com
```

Include:

- affected version or commit
- reproduction steps
- expected and actual behavior
- potential impact
- any suggested mitigation

I will acknowledge valid reports as soon as practical and coordinate a fix before public disclosure.

## Scope

Security-sensitive areas include:

- credential handling
- role assumption
- command output that may expose secrets
- destructive Route 53 operations
- release artifacts and install scripts
