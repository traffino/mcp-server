---
name: mcp-go
description: Go MCP Server Entwicklungspatterns. Offizielles SDK, Tool-Definition, Docker-Build, API-Coverage Tracking.
---

# MCP Go Server Skill

Patterns und Konventionen fuer die Entwicklung von MCP-Servern in Go mit dem offiziellen SDK.

## Rules

| Rule | Impact | Beschreibung |
|------|--------|--------------|
| [server-pattern](rules/server-pattern.md) | CRITICAL | Go MCP Server Bootstrap mit offiziellem SDK |
| [tool-definition](rules/tool-definition.md) | CRITICAL | Tool-Registrierung, Params, Error Handling |
| [docker-build](rules/docker-build.md) | HIGH | Scratch-Image, Multi-stage Build |
| [api-coverage](rules/api-coverage.md) | HIGH | API-Coverage Tracking und Change Detection |
