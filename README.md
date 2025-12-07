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

- 3x Raspberry Pi 5 (Ubuntu Server)
- 1x Switch TP-Link SG105PE (PoE+)
- 1x Adaptador USB-Ethernet

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
