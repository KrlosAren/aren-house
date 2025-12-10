# aren-house

Mi homelab personal - documentación y configuración de toda la infraestructura.

## Arquitectura General

```
Internet
    │
    ▼
  Modem (192.168.1.x)
    │
    │   ┌──────────────────────────────────────────────┐
    │   │              RED HOMELAB                     │
    │   │                                              │
    └───┼── [USB] RPi Gateway [eth0] ─── Switch ───┬── RPi 2
        │        192.168.1.84 │ 10.0.0.1           ├── RPi 3
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
| Raspberry Pi 5 (gateway) | Router/VPN | Fuente dedicada, 2 interfaces de red |
| Raspberry Pi 5 x2 (nodos) | Workers | Alimentados por PoE |
| Switch TP-Link SG105PE | Red interna | 5 puertos Gigabit, 4 PoE+ |
| Adaptador USB-Ethernet | WAN del gateway | Conexión al modem |

## Redes

| Red | Rango | Propósito |
|-----|-------|-----------|
| LAN Casa | 192.168.1.0/24 | Red principal del modem |
| LAN Homelab | 10.0.0.0/24 | Red interna segmentada |
| VPN | 10.0.1.0/24 | Acceso remoto via WireGuard |

## Estado Actual

### Implementado
- [x] Gateway/Router con Raspberry Pi
- [x] Segmentación de red (homelab separado de red principal)
- [x] VPN con WireGuard para acceso remoto
- [x] Automatización con Ansible

### Por hacer
- [ ] Firewall (ufw) con reglas entre redes
- [ ] DHCP server en gateway
- [ ] Docker en todos los nodos
- [ ] Pi-hole para DNS interno
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
| **Componentes** | `{componente}/README.md` | Configuración detallada de cada servicio |
| **Decisiones** | [docs/decisions/](docs/decisions/) | Por qué elegí cada tecnología (ADRs) |

### Decisiones Arquitectónicas (ADRs)

- [001 - WireGuard sobre OpenVPN](docs/decisions/001-wireguard-over-openvpn.md)
- [002 - Segmentación de red con Raspberry Pi](docs/decisions/002-network-segmentation.md)
- [003 - Configuracion de DNS/DHCP/TFTP (dnsmasq)](docs/decisions/003-dnsmasq-dhcp-dns-tftp.md)