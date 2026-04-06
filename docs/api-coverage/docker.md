# Docker Engine API Coverage

- **API**: [Docker Engine API](https://docs.docker.com/engine/api/)
- **API Version**: v1.47 (2026-04)
- **Letzter Check**: 2026-04-06
- **Scope**: read-write

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Containers | GET /containers/json | implemented | list_containers |
| Containers | GET /containers/{id}/json | implemented | inspect_container |
| Containers | GET /containers/{id}/logs | implemented | container_logs |
| Containers | GET /containers/{id}/stats | implemented | container_stats |
| Containers | GET /containers/{id}/top | implemented | container_top |
| Containers | POST /containers/create | implemented | create_container |
| Containers | POST /containers/{id}/start | implemented | start_container |
| Containers | POST /containers/{id}/stop | implemented | stop_container |
| Containers | POST /containers/{id}/restart | implemented | restart_container |
| Containers | DELETE /containers/{id} | implemented | remove_container |
| Containers | POST /containers/{id}/exec | planned | - |
| Images | GET /images/json | implemented | list_images |
| Images | GET /images/{name}/json | implemented | inspect_image |
| Images | POST /images/create | implemented | pull_image |
| Images | DELETE /images/{name} | implemented | remove_image |
| Images | POST /images/prune | implemented | prune_images |
| Images | GET /images/{name}/history | planned | - |
| Networks | GET /networks | implemented | list_networks |
| Networks | GET /networks/{id} | implemented | inspect_network |
| Networks | POST /networks/create | implemented | create_network |
| Networks | DELETE /networks/{id} | implemented | remove_network |
| Networks | POST /networks/{id}/connect | planned | - |
| Networks | POST /networks/{id}/disconnect | planned | - |
| Volumes | GET /volumes | implemented | list_volumes |
| Volumes | GET /volumes/{name} | implemented | inspect_volume |
| Volumes | POST /volumes/create | implemented | create_volume |
| Volumes | DELETE /volumes/{name} | implemented | remove_volume |
| Volumes | POST /volumes/prune | implemented | prune_volumes |
| System | GET /info | implemented | system_info |
| System | GET /version | implemented | system_version |
| System | GET /system/df | implemented | system_df |
| System | GET /_ping | implemented | system_ping |
| System | GET /events | planned | - |

## Hinweise

- Kommunikation ueber Unix Socket (`/var/run/docker.sock`)
- Container-Logs: stdout + stderr, mit Tail und Since Filter
- Stats: `stream=false` fuer einmaligen Snapshot
- geplant: exec, connect/disconnect, events (Streaming)
