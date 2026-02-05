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
│  Storage: SSD 500GB (/srv/nfs, /srv/tftp)                      │
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
│                             │ │  Storage: SSD 240GB         │
│  ┌─────────┐ ┌───────────┐  │ │                             │
│  │ Docker  │ │ Prometheus│  │ │  ┌─────────┐                │
│  │(overlay)│ │  Grafana  │  │ │  │ Docker  │                │
│  └─────────┘ └───────────┘  │ │  │(overlay)│                │
│                             │ │  └─────────┘                │
│  Storage: microSD 29GB      │ │  Storage: SSD 240GB         │
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
| Raspberry Pi 5 | Gateway (rp1-master) | 10.0.0.1 | 100.94.94.49 | SSD 500GB |
| Raspberry Pi 5 | Node (rp2-node) | 10.0.0.2 | via subnet | microSD 29GB |
| Raspberry Pi 5 | Node (rp3-node) | 10.0.0.3 | via subnet | SSD 240GB |

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

## Kubernetes (k3s)

### Arquitectura del Cluster
```
┌─────────────────────────────────────────────────────────────────┐
│                 rp1-master (Control Plane)                       │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     k3s server                            │   │
│  │                                                           │   │
│  │  ┌───────────┐ ┌───────────┐ ┌────────────────────────┐  │   │
│  │  │API Server │ │ Scheduler │ │ Controller Manager     │  │   │
│  │  └───────────┘ └───────────┘ └────────────────────────┘  │   │
│  │                                                           │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌──────────┐  │   │
│  │  │  SQLite   │ │  Flannel  │ │  CoreDNS  │ │ Traefik  │  │   │
│  │  │ (estado)  │ │   (CNI)   │ │   (DNS)   │ │(Ingress) │  │   │
│  │  └───────────┘ └───────────┘ └───────────┘ └──────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  Config crítica: /etc/rancher/k3s/config.yaml                   │
│                  flannel-iface: eth0                             │
└─────────────────────────────────────────────────────────────────┘
                              │
               ┌──────────────┴──────────────┐
               ▼                             ▼
┌───────────────────────────┐ ┌───────────────────────────┐
│   rp2-node (k3s agent)    │ │   rp3-node (k3s agent)    │
│                           │ │                           │
│  ┌─────────────────────┐  │ │  ┌─────────────────────┐  │
│  │ kubelet             │  │ │  │ kubelet             │  │
│  │ kube-proxy          │  │ │  │ kube-proxy          │  │
│  │ containerd          │  │ │  │ containerd          │  │
│  │ MetalLB speaker     │  │ │  │ MetalLB speaker     │  │
│  └─────────────────────┘  │ │  └─────────────────────┘  │
│                           │ │                           │
│  Labels:                  │ │  Labels:                  │
│  - storage=sd             │ │  - storage=ssd            │
│  - storage-size=32gb      │ │  - storage-size=240gb     │
│  (solo workloads          │ │  (workloads con I/O)      │
│   stateless)              │ │                           │
└───────────────────────────┘ └───────────────────────────┘
```

### Redes de Kubernetes

| Red | Rango | Propósito |
|-----|-------|-----------|
| Pods | 10.42.0.0/16 | Red interna de pods |
| Services | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |

### Componentes Instalados

| Componente | Namespace | Función |
|------------|-----------|---------|
| CoreDNS | kube-system | DNS interno del cluster |
| Traefik | kube-system | Ingress Controller |
| MetalLB | metallb-system | LoadBalancer para bare-metal |
| local-path-provisioner | kube-system | Storage dinámico |

### DNS y Acceso a Servicios
```
┌─────────────────────────────────────────────────────────────┐
│                        dnsmasq                               │
│                                                              │
│  *.homelab.local      → 10.0.0.1   (Traefik Docker)         │
│  *.k8s.homelab.local  → 10.0.0.50  (Traefik k3s/MetalLB)    │
└─────────────────────────────────────────────────────────────┘
```

### Flujo de Tráfico k8s
```
Cliente (Mac/LAN)
       │
       │ http://app.k8s.homelab.local
       ▼
┌─────────────────────┐
│  dnsmasq            │
│  Resuelve a         │
│  10.0.0.50          │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  MetalLB            │
│  Anuncia IP via ARP │
│  Dirige al nodo     │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Traefik (k3s)      │
│  LoadBalancer       │
│  :80/:443           │
│                     │
│  Rutea por Host     │
│  header al Ingress  │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Service            │
│  (ClusterIP)        │
└──────────┬──────────┘
           ▼
┌─────────────────────┐
│  Pod                │
└─────────────────────┘
```

### Flannel (CNI)

Red overlay que permite comunicación entre pods en diferentes nodos:
```
Pod en rp3 (10.42.3.x)
       │
       ▼
┌─────────────────────┐
│  flannel.1 (VXLAN)  │
│  Encapsula paquete  │
└──────────┬──────────┘
           │ UDP 8472
           ▼
┌─────────────────────┐
│  eth0 (10.0.0.3)    │
│  Red física         │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  eth0 (10.0.0.1)    │
│  Red física         │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  flannel.1 (VXLAN)  │
│  Desencapsula       │
└──────────┬──────────┘
           ▼
Pod en rp1 (10.42.0.x)
```

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
| MetalLB para LoadBalancer | Asigna IPs reales en bare-metal | [ADR-011](decisions/011-metallb.md) |
| k3s sobre k8s vanilla | Menor uso de recursos, ideal para ARM/homelab | [ADR-012](decisions/012-k3s-over-k8s.md) |

---

## Expansión Futura

### Agregar más nodos

1. Obtener serial number de la nueva Pi
2. Ejecutar `prepare-node.yml`
3. Configurar DHCP en dnsmasq
4. El nodo bootea automáticamente por red

### Kubernetes - Próximos pasos

- [ ] **Longhorn**: Storage distribuido con replicación
- [ ] **Cert-Manager**: Certificados TLS automáticos
- [ ] **Observability en k8s**: Migrar Prometheus/Grafana/Loki al cluster
- [ ] **Alertmanager**: Alertas

### Storage distribuido

Opciones:
- Longhorn (replica datos entre nodos)
- NFS desde gateway (actual)
- Ceph (más complejo)
