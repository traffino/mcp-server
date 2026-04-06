# API Coverage Tracking

Jeder MCP-Server hat ein `docs/api-coverage/<server>.md` das den Implementierungsstand dokumentiert.

## Format

```markdown
# <Server> API Coverage

- **API**: <Name und Link zur offiziellen Doku>
- **API Version**: <Version oder Datum der API-Spec>
- **Letzter Check**: <YYYY-MM-DD>
- **Scope**: readonly | read-write | proxy

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Servers | GET /servers | implemented | list_servers |
| Servers | GET /servers/{id} | implemented | get_server |
| Servers | POST /servers | out-of-scope | - |
```

## Status-Werte

- `implemented` — Endpoint ist als MCP-Tool verfuegbar
- `planned` — Wird noch implementiert
- `out-of-scope` — Bewusst nicht implementiert (z.B. schreibende Ops bei readonly-Servern)

## Regeln

- Jeder implementierte Endpoint hat einen `Tool-Name`
- `out-of-scope` Eintraege dokumentieren WARUM sie ausgeschlossen sind (z.B. "readonly scope")
- API-Version und Letzter-Check-Datum muessen aktuell gehalten werden
- Bei API-Aenderungen: neuen Eintrag im Changelog-Abschnitt
- OpenAPI/Spec-Link wenn verfuegbar

## Change Detection

Wenn die offizielle API-Dokumentation neue Endpoints hat:
1. `planned` Eintraege hinzufuegen
2. Issue erstellen fuer die Implementierung
3. api-coverage Dokument aktualisieren
