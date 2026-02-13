# OpenClaw Go

A lightweight Go rewrite of [OpenClaw](https://github.com/openclaw/openclaw) — focused on lower memory footprint for running multiple AI agents on resource-constrained hardware.

## Why

The original OpenClaw runs on Node.js. Running 7+ agent instances consumes significant memory. A Go rewrite targets:
- ~10-20MB per agent instance (vs ~100-200MB for Node.js)
- Single binary deployment
- Native concurrency for agent orchestration
- Lower latency for tool execution

## Team

This project is built by Eva's engineering team:

| Agent | Role | Focus |
|-------|------|-------|
| **Eva** | Project Manager | Architecture, coordination, PRs |
| **Marcus** | Code Reviewer | All PRs reviewed before merge |
| **Iris** | Researcher | OpenClaw internals, API docs |
| **Sentinel** | Security | Auth, credential handling, sandboxing |
| **Bolt** | Embedded/Low-level | Serial, networking, system calls |
| **Atlas** | Infrastructure | CI/CD, Docker, deployment |
| **Pixel** | QA | Tests, integration testing, edge cases |

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full design.

## Status

🚧 **Pre-alpha** — Architecture phase. Not functional yet.

## License

MIT
