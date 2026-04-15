# URLs del Homelab

## Servicios k8s

| URL | Servicio | Namespace | Puerto |
|-----|----------|-----------|--------|
| `prometheus.k8s.homelab.local` | Prometheus | monitoring | 9090 |
| `grafana.k8s.homelab.local` | Grafana | monitoring | 3000 |
| `registry.k8s.homelab.local` | Docker Registry v2 | registry | 5000 |
| `registry-ui.k8s.homelab.local` | Registry UI | registry | 80 |
| `n8n.k8s.homelab.local` | n8n | n8n-system | 5678 |
| `test-app.k8s.homelab.local` | Test App | simple-app | 3000 |
| `longhorn.k8s.homelab.local` | Longhorn UI | longhorn-system | 80 |

Todos los servicios k8s se acceden via:
```
Cliente → DNS → 192.168.1.89 → DNAT → 10.0.0.50 (MetalLB) → Traefik k3s → Ingress → Service
```

## Servicios Docker (legacy)

| URL | Servicio |
|-----|----------|
| `*.homelab.local` | Resuelve a 10.0.0.1 (Traefik Docker en rp1-master) |

## Acceso directo

| URL/IP | Servicio |
|--------|----------|
| `10.0.0.1:5432` | PostgreSQL (Docker en rp1-master) |
| `10.0.0.1:9100` | Node Exporter (rp1-master) |
| `10.0.0.2:9100` | Node Exporter (rp2-node) |
| `10.0.0.3:9100` | Node Exporter (rp3-node) |

## Configuración DNS

- `*.homelab.local` → `10.0.0.1` (dnsmasq, host-record)
- `*.k8s.homelab.local` → `192.168.1.89` (dnsmasq, address wildcard + DNAT)

Ver [dns-setup.md](../dns-setup.md) para configuración detallada.
