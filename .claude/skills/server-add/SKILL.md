---
name: server-add
description: Use when adding a new MCP server to the monorepo. Covers all required files, CI config, infrastructure integration, and documentation updates.
---

# Neuen MCP-Server hinzufuegen

Checkliste fuer das Hinzufuegen eines neuen MCP-Servers zum Monorepo — inklusive aller Dateien, die oft vergessen werden.

## Server-Typen

| Typ | Beispiele | Beschreibung |
|-----|-----------|-------------|
| MCP Server | brave, github, hetzner | Eigene Tool-Implementierung mit offiziellem SDK |
| HTTP Proxy | cloudflare, drawio | Leitet Requests an Remote-MCP-Server weiter |
| Externes Binary | memory | Nur Dockerfile, kein eigener Go-Code |

## Checkliste

### 1. Go-Code

- [ ] `cmd/<name>/main.go` — Server-Binary (siehe Skill `mcp-go` fuer Patterns)
- [ ] Build testen: `make <name>`

Fuer HTTP-Proxies: `cmd/cloudflare/main.go` oder `cmd/drawio/main.go` als Vorlage.
Fuer MCP-Server: `cmd/brave/main.go` als Vorlage.

### 2. Dockerfile

- [ ] `docker/<name>.Dockerfile`
- [ ] Build testen: `make docker-<name>`

Siehe Skill `mcp-go` Rule `docker-build` fuer das Standard-Pattern.

### 3. CI/CD — GitHub Actions

- [ ] `.github/workflows/docker.yml` — Server zur Build-Matrix hinzufuegen

```yaml
matrix:
  server: [brave, bunq, cloudflare, ..., <name>, ..., memory]
```

Alphabetisch einsortieren. Ohne diesen Schritt wird kein Docker-Image gebaut und gepusht.

### 4. Dokumentation (dieses Repo)

- [ ] `docs/api-coverage/<name>.md` — Tools und Parameter dokumentieren
- [ ] `CLAUDE.md` — Eintrag in der Server-Uebersicht-Tabelle

### 5. Infrastructure (traffino/infrastructure-home)

- [ ] `docker-compose.yml` — Service-Definition hinzufuegen
- [ ] `docker-compose.yml` — Aggregator `MCP_BACKENDS` erweitern (Personal und/oder Baunach)
- [ ] `docker-compose.yml` — Aggregator `depends_on` erweitern
- [ ] `CLAUDE.md` — Backend-Listen und Backend-Typen-Tabelle aktualisieren

#### Shared vs. Profil-spezifisch

| Kriterium | Shared (wie brave, drawio) | Profil-spezifisch (wie github, discord) |
|-----------|---------------------------|----------------------------------------|
| Credentials | Keine oder identisch | Unterschiedlich pro Profil |
| Container-Name | `ai-mcp-<name>` | `ai-mcp-<name>-personal`, `ai-mcp-<name>-baunach` |
| Netzwerke | Beide (`ai-mcp-personal` + `ai-mcp-baunach`) | Nur eigenes Profil-Netzwerk |
| Instanzen | 1 | 2 (eine pro Stack) |

#### Service-Definition (Shared)

```yaml
  ai-mcp-<name>:
    image: traffino/mcp-<name>:latest
    container_name: ai-mcp-<name>
    restart: unless-stopped
    mem_limit: 64m
    networks:
      - ai-mcp-personal
      - ai-mcp-baunach
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8000/health"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 5s
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### Aggregator-Einbindung

In `MCP_BACKENDS` den neuen Service anhaengen:

```
,<name>=http://ai-mcp-<name>:8000/mcp
```

In `depends_on` den neuen Service hinzufuegen:

```yaml
      ai-mcp-<name>:
        condition: service_healthy
```

## Haeufige Fehler

| Fehler | Auswirkung |
|--------|-----------|
| GitHub Actions Matrix vergessen | Kein Docker-Image auf Docker Hub |
| Aggregator `depends_on` vergessen | Aggregator startet vor Backend → Tools fehlen |
| Nur einen Aggregator aktualisiert | Server fehlt im zweiten Stack |
| infrastructure-home CLAUDE.md vergessen | Naechste Session kennt den neuen Server nicht |
