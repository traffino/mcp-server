# Discord Bot API Coverage

- **API**: [Discord API v10](https://discord.com/developers/docs)
- **API Version**: v10 (2026-04)
- **Letzter Check**: 2026-04-06
- **Scope**: read-write (teilweise)

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Guilds | GET /users/@me/guilds | implemented | list_guilds |
| Guilds | GET /guilds/{id} | implemented | get_guild |
| Guilds | GET /guilds/{id}/channels | implemented | list_guild_channels |
| Guilds | GET /guilds/{id}/members | implemented | list_guild_members |
| Channels | GET /channels/{id} | implemented | get_channel |
| Channels | POST /guilds/{id}/channels | implemented | create_channel |
| Channels | PATCH /channels/{id} | implemented | edit_channel |
| Roles | GET /guilds/{id}/roles | implemented | list_roles |
| Reactions | GET /channels/{id}/messages/{id}/reactions/{emoji} | implemented | list_reactions |
| Reactions | PUT /channels/{id}/messages/{id}/reactions/{emoji}/@me | implemented | add_reaction |
| Reactions | DELETE /channels/{id}/messages/{id}/reactions/{emoji}/@me | implemented | remove_reaction |
| Threads | GET /guilds/{id}/threads/active | implemented | list_active_threads |
| Threads | GET /channels/{id}/threads/archived/{type} | implemented | list_archived_threads |
| Threads | POST /channels/{id}/threads | implemented | create_thread |
| Threads | PUT /channels/{id}/thread-members/@me | implemented | join_thread |
| Users | GET /users/{id} | implemented | get_user |
| Users | GET /users/@me | implemented | get_current_user |
| Messages | GET /channels/{id}/messages | out-of-scope | - |
| Messages | POST /channels/{id}/messages | out-of-scope | - |
| Voice | GET /voice/regions | out-of-scope | - |
| Webhooks | GET /channels/{id}/webhooks | out-of-scope | - |

## Hinweise

- Messages senden/lesen, Voice und Webhooks sind out-of-scope
- Auth: `Authorization: Bot` Header
- Rate Limits: Discord hat strenge Rate Limits (siehe API Docs)
