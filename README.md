# aren-house

Mi homelab personal - documentación y configuración de toda la infraestructura.

## Arquitectura General

```
Internet (CGNAT)
    │
    ▼
  Modem (192.168.100.x)
    │
    │ [USB-ETH] enx00e04c683da2
    │
    ▼
  rp1-master (Gateway + k3s Control Plane)
  192.168.100.x WAN / 10.0.0.1 LAN
    │
    │ [eth0] LAN 10.0.0.0/24
    │
    ▼
  Switch TP-Link SG105PE (10.0.0.5)
    │
    ├── rp2-node (10.0.0.2) - Netboot, k3s worker, microSD 32GB
    └── rp3-node (10.0.0.3) - Netboot, k3s worker, SSD 240GB

  Acceso remoto: Tailscale VPN (100.x.x.x, mesh, bypasses CGNAT)
```

## Componentes

| Componente | Descripción | Documentación |
|------------|-------------|---------------|
| **Ansible** | Automatización y configuración de infraestructura | [homelab-ansible/README.md](homelab-ansible/README.md) |

## Hardware

| Dispositivo | Rol | Notas |
|-------------|-----|-------|
| Raspberry Pi 5 (rp1-master) | Gateway/Router/VPN | Fuente dedicada, 2 interfaces de red |
| Raspberry Pi 5 x2 (rp2, rp3) | Workers | Netboot via PXE/NFS (sin microSD), PoE |
| Switch TP-Link SG105PE | Red interna | 5 puertos Gigabit, 4 PoE+ |
| Adaptador USB-Ethernet | WAN del gateway | Conexión al modem |

## Redes

| Red | Rango | Propósito |
|-----|-------|-----------|
| WAN (Modem) | 192.168.100.0/24 | Red del modem (DHCP) |
| LAN Homelab | 10.0.0.0/24 | Red interna segmentada |
| Pods (k8s) | 10.42.0.0/16 | Red interna de pods |
| Services (k8s) | 10.43.0.0/16 | ClusterIPs |
| MetalLB | 10.0.0.50-60 | LoadBalancer IPs |
| DHCP | 10.0.0.100-200 | Clientes DHCP |
| Tailscale | 100.x.x.x | VPN mesh (bypasses CGNAT) |

## Estado Actual

### Implementado
- [x] Gateway/Router con Raspberry Pi
- [x] Segmentación de red (homelab separado de red principal)
- [x] Automatización con Ansible
- [x] DHCP/DNS/TFTP con dnsmasq
- [x] DNS local (.homelab.local, .k8s.homelab.local)
- [x] Netboot (PXE/NFS) para nodos rp2 y rp3
- [x] NAT/IP forwarding para salida a internet
- [x] Firewall (UFW) con reglas entre redes
- [x] Docker en nodos con storage local (overlay2)
- [x] Tailscale VPN (reemplazó WireGuard por CGNAT)
- [x] DuckDNS para DNS dinámico
- [x] k3s cluster (3 nodos)
- [x] MetalLB para LoadBalancer (10.0.0.50-60)
- [x] Monitoreo con Prometheus/Grafana/node_exporter

### Por hacer
- [ ] Observability en k8s (migrar Prometheus/Grafana/Loki al cluster)
- [ ] Longhorn (storage distribuido)
- [ ] Cert-Manager (certificados TLS)
- [ ] Alertmanager

## Inicio Rápido

```bash
# Clonar el repositorio
git clone git@github.com:KrlosAren/aren-house.git
cd aren-house

# Configurar Ansible
cd homelab-ansible
ansible all -m ping

# Desplegar configuración del gateway
ansible-playbook playbooks/gateway.yml
```

Para más detalles, ver la documentación de cada componente.

## Documentación

| Tipo | Ubicación | Descripción |
|------|-----------|-------------|
| **Decisiones** | [docs/decisions/](docs/decisions/) | ADRs - Por qué elegí cada tecnología |
| **Conceptos** | [docs/concepts/](docs/concepts/) | Teoría: DHCP, DNS, PXE, NAT, NFS, etc. |
| **Guías** | [docs/guides/](docs/guides/) | How-to: playbooks, firewall, troubleshooting |
| **Runbooks** | [docs/runbooks/](docs/runbooks/) | Procedimientos: disaster-recovery, maintenance |
| **Ansible** | [homelab-ansible/README.md](homelab-ansible/README.md) | Automatización de infraestructura |

### Decisiones Arquitectónicas (ADRs)

- [001 - WireGuard sobre OpenVPN](docs/decisions/001-wireguard-over-openvpn.md)
- [002 - Segmentación de red con Raspberry Pi](docs/decisions/002-network-segmentation.md)
- [003 - Configuracion de DNS/DHCP/TFTP (dnsmasq)](docs/decisions/003-dnsmasq-dhcp-dns-tftp.md)
- [004 - IP Forwarding y NAT](docs/decisions/004-ip-forwarding-nat.md)
- [005 - UFW - Firewall](docs/decisions/005-ufw-firewall.md)
- [006 - Netboot vs Local](docs/decisions/006-netboot-vs-local.md)
- [007 - Docker storage Overlay](docs/decisions/007-docker-storage-overlay.md)
- [008 - Tailscale CGNAT](docs/decisions/008-tailscale-cgnat.md)
- [009 - CGNAT Workaround](docs/decisions/009-cgnat-workaround.md)
- [010 - K3s Storage en Discos Locales](docs/decisions/010-k3s-storage-on-nfs.md)
- [011 - MetalLB para LoadBalancer](docs/decisions/011-metallb.md)
- [012 - K3s sobre Kubernetes Vanilla](docs/decisions/012-k3s-over-k8s.md)