# Brave Search API Coverage

- **API**: [Brave Search API](https://api-dashboard.search.brave.com/app/documentation/web-search/query)
- **API Version**: v1 (2026-04)
- **Letzter Check**: 2026-04-06
- **Scope**: readonly

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Web Search | GET /res/v1/web/search | implemented | web_search |
| Suggest | GET /res/v1/suggest/search | implemented | suggest |
| News Search | GET /res/v1/news/search | out-of-scope | - |
| Video Search | GET /res/v1/videos/search | out-of-scope | - |
| Image Search | GET /res/v1/images/search | out-of-scope | - |
| Summarizer | GET /res/v1/summarizer/search | out-of-scope | - |
| Local POIs | GET /res/v1/local/pois | out-of-scope | - |
| Local Descriptions | GET /res/v1/local/descriptions | out-of-scope | - |

### web_search Parameters

| Parameter | Typ | Implementiert |
|-----------|-----|---------------|
| q (query) | string | ja |
| count | int | ja |
| offset | int | ja |
| country | string | ja |
| search_lang | string | ja |
| freshness | string | ja |
| safesearch | string | ja |
| text_decorations | bool | nein (default) |
| spellcheck | bool | nein (default) |
| result_filter | string | nein (default) |
| extra_snippets | bool | nein |
| summary | bool | nein (premium) |
| goggles | string[] | nein |

### suggest Parameters

| Parameter | Typ | Implementiert |
|-----------|-----|---------------|
| q (query) | string | ja |
| country | string | ja |
| count | int | ja |
| rich | bool | ja |

## Hinweise

- Free Tier: $5 monatliches Guthaben (~1000 Queries)
- Auth: `X-Subscription-Token` Header
- Responses sind gzip-komprimiert
- out-of-scope Endpoints: Fokus auf Web Search + Suggest fuer allgemeine Suche
