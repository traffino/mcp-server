# Hetzner Cloud API Coverage

- **API**: [Hetzner Cloud API](https://docs.hetzner.cloud/)
- **API Version**: v1 (2026-04)
- **Letzter Check**: 2026-04-06
- **Scope**: readonly

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Servers | GET /servers | implemented | list_servers |
| Servers | GET /servers/{id} | implemented | get_server |
| Servers | GET /servers/{id}/metrics | implemented | get_server_metrics |
| Servers | POST /servers | out-of-scope | - |
| Servers | DELETE /servers/{id} | out-of-scope | - |
| Servers | POST /servers/{id}/actions/* | out-of-scope | - |
| Server Types | GET /server_types | implemented | list_server_types |
| SSH Keys | GET /ssh_keys | implemented | list_ssh_keys |
| SSH Keys | GET /ssh_keys/{id} | implemented | get_ssh_key |
| SSH Keys | POST/PUT/DELETE | out-of-scope | - |
| Firewalls | GET /firewalls | implemented | list_firewalls |
| Firewalls | GET /firewalls/{id} | implemented | get_firewall |
| Firewalls | POST/PUT/DELETE | out-of-scope | - |
| Networks | GET /networks | implemented | list_networks |
| Networks | GET /networks/{id} | implemented | get_network |
| Networks | POST/PUT/DELETE | out-of-scope | - |
| Volumes | GET /volumes | implemented | list_volumes |
| Volumes | GET /volumes/{id} | implemented | get_volume |
| Volumes | POST/PUT/DELETE | out-of-scope | - |
| Floating IPs | GET /floating_ips | implemented | list_floating_ips |
| Floating IPs | GET /floating_ips/{id} | implemented | get_floating_ip |
| Floating IPs | POST/PUT/DELETE | out-of-scope | - |
| Images | GET /images | implemented | list_images |
| Images | GET /images/{id} | implemented | get_image |
| Images | DELETE | out-of-scope | - |
| Locations | GET /locations | implemented | list_locations |
| Locations | GET /locations/{id} | implemented | get_location |
| Datacenters | GET /datacenters | implemented | list_datacenters |
| Datacenters | GET /datacenters/{id} | implemented | get_datacenter |
| Load Balancers | GET /load_balancers | implemented | list_load_balancers |
| Load Balancers | GET /load_balancers/{id} | implemented | get_load_balancer |
| Load Balancer Types | GET /load_balancer_types | implemented | list_load_balancer_types |
| Load Balancers | POST/PUT/DELETE | out-of-scope | - |
| Certificates | GET /certificates | implemented | list_certificates |
| Certificates | GET /certificates/{id} | implemented | get_certificate |
| Certificates | POST/DELETE | out-of-scope | - |
| Primary IPs | GET /primary_ips | implemented | list_primary_ips |
| Primary IPs | GET /primary_ips/{id} | implemented | get_primary_ip |
| Primary IPs | POST/PUT/DELETE | out-of-scope | - |
| Placement Groups | GET /placement_groups | implemented | list_placement_groups |
| Placement Groups | GET /placement_groups/{id} | implemented | get_placement_group |
| Placement Groups | POST/DELETE | out-of-scope | - |
| Actions | GET /actions | implemented | list_actions |
| Actions | GET /actions/{id} | implemented | get_action |
| Pricing | GET /pricing | implemented | get_pricing |

## Hinweise

- Alle schreibenden Endpoints sind out-of-scope (readonly)
- Auth: `Authorization: Bearer` Header
- Pagination: `page`, `per_page` (max 50), `sort`
- Filter: `name` fuer die meisten List-Endpoints
