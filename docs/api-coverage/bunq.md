# Bunq Banking API Coverage

- **API**: [Bunq API](https://doc.bunq.com/)
- **API Version**: v1 (2026-04)
- **Letzter Check**: 2026-04-06
- **Scope**: readonly

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Accounts | GET /user/{id}/monetary-account | implemented | list_accounts |
| Accounts | GET /user/{id}/monetary-account/{id} | implemented | get_account |
| Payments | GET /user/{id}/monetary-account/{id}/payment | implemented | list_payments |
| Payments | GET /user/{id}/monetary-account/{id}/payment/{id} | implemented | get_payment |
| Payments | POST (create payment) | out-of-scope | - |
| Cards | GET /user/{id}/card | implemented | list_cards |
| Cards | GET /user/{id}/card/{id} | implemented | get_card |
| Schedules | GET /user/{id}/monetary-account/{id}/schedule | implemented | list_schedules |
| Schedules | GET /user/{id}/monetary-account/{id}/schedule/{id} | implemented | get_schedule |
| Schedules | POST (create) | out-of-scope | - |
| Requests | GET /user/{id}/monetary-account/{id}/request-inquiry | out-of-scope | - |
| Draft Payments | GET /user/{id}/monetary-account/{id}/draft-payment | out-of-scope | - |
| Invoices | GET /user/{id}/invoice | out-of-scope | - |
| Events | GET /user/{id}/event | out-of-scope | - |

## Hinweise

- Nur Leseoperationen implementiert
- Auth: `X-Bunq-Client-Authentication` Header
- Bunq API erfordert Session-Token (nicht API-Key direkt)
- User-Agent Header empfohlen
