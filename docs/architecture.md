# Arquitectura del Homelab

## Visión General
```
                              INTERNET
                                  │
                                  │ CGNAT (ISP Entel)
                                  │
                         ┌────────┴────────┐
                         │     Modem       │
                         │  192.168.100.1  │
                         └────────┬────────┘
                                  │
                                  │ 192.168.100.18
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                        rp1-master                               │
│                         GATEWAY                                 │
│                        10.0.0.1                                 │
│                                                                 │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────────┐ │
│  │ dnsmasq │ │   NFS   │ │  TFTP   │ │   ufw   │ │ Tailscale │ │
│  │  DHCP   │ │ Server  │ │ Server  │ │Firewall │ │    VPN    │ │
│  │   DNS   │ │         │ │         │ │         │ │           │ │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └───────────┘ │
│                                                                 │
│  Storage: SSD 250GB (/srv/nfs, /srv/tftp)                      │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  │ eth0 (LAN 10.0.0.0/24)
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
                    ▼                           ▼
┌─────────────────────────────┐ ┌─────────────────────────────┐
│         rp2-node            │ │         rp3-node            │
│         10.0.0.2            │ │         10.0.0.3            │
│                             │ │                             │
│  Boot: NFS (desde gateway)  │ │  Boot: NFS (desde gateway)  │
│  Docker: microSD local      │ │  Docker: SSD local          │
│                             │ │  Storage: SSD 433GB         │
│  ┌─────────┐ ┌───────────┐  │ │                             │
│  │ Docker  │ │ Prometheus│  │ │  ┌─────────┐                │
│  │(overlay)│ │  Grafana  │  │ │  │ Docker  │                │
│  └─────────┘ └───────────┘  │ │  │(overlay)│                │
│                             │ │  └─────────┘                │
│  Storage: microSD 29GB      │ │  Storage: SSD 500GB         │
└─────────────────────────────┘ └─────────────────────────────┘


                    ACCESO REMOTO (via Tailscale)

┌─────────────────┐         ┌─────────────────┐
│    Tu Mac       │◄───────►│ Tailscale Cloud │
│  100.70.50.39   │         │   (coordina)    │
└─────────────────┘         └────────┬────────┘
                                     │
                                     ▼
                            ┌─────────────────┐
                            │   rp1-master    │
                            │  100.94.94.49   │
                            │ (subnet router) │
                            └─────────────────┘
```

---

## Componentes

### Hardware

| Dispositivo | Rol | IP LAN | IP Tailscale | Storage |
|-------------|-----|--------|--------------|---------|
| Raspberry Pi 5 | Gateway (rp1-master) | 10.0.0.1 | 100.94.94.49 | SSD 250GB |
| Raspberry Pi 5 | Node (rp2-node) | 10.0.0.2 | via subnet | microSD 29GB |
| Raspberry Pi 5 | Node (rp3-node) | 10.0.0.3 | via subnet | SSD 500GB |

### Redes

| Red | Rango | Propósito |
|-----|-------|-----------|
| WAN | 192.168.100.0/24 | Conexión al modem/internet |
| LAN | 10.0.0.0/24 | Red interna del homelab |
| VPN | 10.0.1.0/24 | Clientes Tailscale |
| Tailscale | 100.x.x.x | Mesh VPN |

### Servicios en Gateway

| Servicio | Puerto | Función |
|----------|--------|---------|
| dnsmasq | 53, 67-68 | DNS y DHCP |
| NFS | 2049, 111 | Filesystem para netboot |
| TFTP | 69 | Boot files para netboot |
| Tailscale | - | VPN mesh |
| DuckDNS | - | DNS dinámico |

### Servicios en Nodos

| Servicio | Nodo | Puerto | Función |
|----------|------|--------|---------|
| Docker | rp2, rp3 | - | Contenedores |
| Prometheus | rp2 | 9090 | Métricas |
| Grafana | rp2 | 3000 | Dashboards |
| node_exporter | todos | 9100 | Métricas del sistema |

---

## Flujos de Red

### Boot de un nodo
```
rp2-node enciende
    │
    ├─► DHCP Request (broadcast)
    │       │
    │       ▼
    │   rp1-master (dnsmasq)
    │       │
    │       ▼
    │   DHCP Response: IP=10.0.0.2, TFTP=10.0.0.1
    │
    ├─► TFTP: Descarga kernel y initrd
    │       │
    │       ▼
    │   /srv/tftp/{serial}/
    │
    └─► NFS: Monta root filesystem
            │
            ▼
        /srv/nfs/rp2/
```

### Acceso remoto (desde internet)
```
Tu Mac (cualquier red)
    │
    ├─► Tailscale client
    │       │
    │       ▼ (conexión saliente, atraviesa CGNAT)
    │   Tailscale Cloud (coordina)
    │       │
    │       ▼
    │   rp1-master (Tailscale, subnet router)
    │       │
    │       ▼
    └─► Acceso a 10.0.0.0/24
            │
            ├── rp1: 10.0.0.1
            ├── rp2: 10.0.0.2
            └── rp3: 10.0.0.3
```

### Monitoreo
```
node_exporter (todos los nodos)
    │
    │ :9100/metrics
    │
    ▼
Prometheus (rp2:9090)
    │
    │ scrape cada 15s
    │
    ▼
Grafana (rp2:3000)
    │
    │ queries
    │
    ▼
Dashboards
```

---

## Decisiones de Diseño

| Decisión | Razón | ADR |
|----------|-------|-----|
| Netboot para nodos | Gestión centralizada, fácil reinstalación | [ADR-006](decisions/006-netboot-vs-local.md) |
| Storage local para Docker | overlay2 no funciona sobre NFS | [ADR-007](decisions/007-docker-storage-overlay.md) |
| Tailscale en lugar de WireGuard | CGNAT del ISP bloquea conexiones entrantes | [ADR-008](decisions/008-tailscale-cgnat.md) |
| Workaround CGNAT | ISP usa CGNAT, port forwarding no funciona | [ADR-009](decisions/009-cgnat-workaround.md) |
| Segmentación de red | Aislamiento LAN interna del modem | [ADR-002](decisions/002-network-segmentation.md) |
| dnsmasq como DHCP/DNS/TFTP | Solución integrada y ligera | [ADR-003](decisions/003-dnsmasq-dhcp-dns-tftp.md) |
| UFW como firewall | Simplicidad sobre iptables directo | [ADR-005](decisions/005-ufw-firewall.md) |

---

## Expansión Futura

### Agregar más nodos

1. Obtener serial number de la nueva Pi
2. Ejecutar `prepare-node.yml`
3. Configurar DHCP en dnsmasq
4. El nodo bootea automáticamente por red

### Migrar a Kubernetes (k3s)
```
rp1-master: k3s server (control plane)
rp2-node:   k3s agent (worker)
rp3-node:   k3s agent (worker)
```

### Storage distribuido

Opciones:
- Longhorn (replica datos entre nodos)
- NFS desde gateway (actual)
- Ceph (más complejo)
