# aren-house

Mi homelab personal - documentación y configuración de toda la infraestructura.

## Arquitectura General

```
Internet
    │
    ▼
  Modem (192.168.100.x)
    │
    │   ┌──────────────────────────────────────────────┐
    │   │              RED HOMELAB                     │
    │   │                                              │
    └───┼── [USB] RPi Gateway [eth0] ─── Switch ───┬── RPi 2 (netboot)
        │        WAN DHCP     │ 10.0.0.1           ├── RPi 3 (netboot)
        │                     │                    └── (expansión)
        │                     │                         10.0.0.x
        │   ┌─────────────────┘
        │   │ WireGuard VPN
        │   │ 10.0.1.0/24
        │   │
        └───┼─────────────────────────────────────────┘
            │
     Mac ───┘ (10.0.1.2 via VPN)
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
| VPN | 10.0.1.0/24 | Acceso remoto via WireGuard |

## Estado Actual

### Implementado
- [x] Gateway/Router con Raspberry Pi
- [x] Segmentación de red (homelab separado de red principal)
- [x] VPN con WireGuard para acceso remoto
- [x] Automatización con Ansible
- [x] DHCP/DNS/TFTP con dnsmasq
- [x] DNS local (.homelab.local)
- [x] Netboot (PXE/NFS) para nodos rp2 y rp3
- [x] NAT/IP forwarding para salida a internet

### Por hacer
- [ ] Firewall (ufw) con reglas entre redes
- [ ] Docker en nodos
- [ ] k3s cluster
- [ ] Monitoreo con Prometheus/Grafana

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