# Homelab Ansible

Configuración automatizada de mi homelab con Raspberry Pi usando Ansible.

## ¿Por qué este proyecto?

### Segmentación de red

Decidí no conectar mi homelab directamente al modem. En su lugar, una Raspberry Pi actúa como **gateway/router**, creando una red interna separada. Esto me permite:

- Experimentar con segmentación de redes sin afectar la red principal de casa
- Aprender sobre routing, NAT y firewalls en un entorno real
- Tener control total sobre el tráfico entre mi homelab e internet

> **Nota:** Segmentación no es igual a seguridad. Tener redes separadas es el primer paso, pero para seguridad real se necesitan reglas de firewall que controlen qué tráfico puede pasar entre redes.

### Infraestructura como código

Uso Ansible para automatizar toda la configuración porque:

- **Recuperación ante desastres**: Ya perdí toda mi configuración una vez. Nunca más. Con Ansible puedo reconstruir todo desde cero en minutos.
- **Documentación viva**: El código ES la documentación. Cada playbook explica exactamente cómo está configurado cada servicio.
- **Conocimiento acumulado**: Cada cosa nueva que aprendo queda guardada como código, no solo en mi memoria.

### VPN con WireGuard

WireGuard crea un túnel encriptado que me permite acceder a la red interna del homelab desde mi Mac. En lugar de:

1. Conectar SSH al gateway (192.168.1.84)
2. Desde ahí, conectar SSH a la máquina interna (10.0.0.x)

Con WireGuard:

1. Conecto la VPN
2. Accedo directamente a 10.0.0.x

El "salto" sigue existiendo, pero WireGuard lo hace transparente creando una interfaz virtual en mi Mac que simula estar conectado directamente al switch.

### PXE Boot (Netboot)

Los nodos worker (rp2, rp3) bootean desde la red, sin necesidad de microSD. El gateway (rp1-master) sirve:

- **DHCP**: Asigna IPs fijas por MAC address
- **TFTP**: Sirve kernel y archivos de boot
- **NFS**: Proporciona el root filesystem

Esto permite gestionar los nodos de forma centralizada y facilita la recuperación ante fallos.

## Arquitectura
```
Internet
    │
    ▼
  Modem (192.168.1.x)
    │
    │   ┌──────────────────────────────────────────────┐
    │   │              RED HOMELAB                     │
    │   │                                              │
    └───┼── [USB] RPi Gateway [eth0] ─── Switch ───┬── RPi 2 (netboot)
        │        192.168.1.84 │ 10.0.0.1           ├── RPi 3 (netboot)
        │                     │                    └── 10.0.0.x
        │   ┌─────────────────┘
        │   │ WireGuard VPN
        │   │ 10.0.1.0/24
        │   │
        └───┼─────────────────────────────────────────┘
            │
     Mac ───┘ (10.0.1.2 via VPN)
```

## Dispositivos

| Dispositivo | IP | MAC | Rol |
|-------------|-----|-----|-----|
| rp1-master | 10.0.0.1 | 2c:cf:67:a9:b8:51 | Gateway, DHCP, DNS, TFTP, NFS |
| rp2 | 10.0.0.2 | 2c:cf:67:88:9e:f5 | Nodo worker (netboot) |
| rp3 | 10.0.0.3 | 2c:cf:67:a9:b9:13 | Nodo worker (netboot) |
| switch | 10.0.0.5 | ec:75:0c:ff:fc:d6 | TP-Link SG105PE |

## Redes

| Red | Rango | Propósito |
|-----|-------|-----------|
| LAN Casa | 192.168.1.0/24 | Red principal del modem |
| LAN Homelab | 10.0.0.0/24 | Red interna segmentada |
| VPN | 10.0.1.0/24 | Acceso remoto via WireGuard |

## Requisitos

- Ansible instalado en tu máquina local
- SSH con clave pública configurado
- Raspberry Pi con Ubuntu Server

### Instalar Ansible (macOS)
```bash
brew install ansible
```

## Configuración inicial del Raspberry Pi

Después de instalar Ubuntu Server en cada Raspberry Pi, hay pasos críticos de seguridad antes de usar Ansible.

### 1. Generar clave SSH (en tu Mac, si no tienes una)
```bash
ls ~/.ssh/id_ed25519.pub  # verificar si existe

# Si no existe, generar:
ssh-keygen -t ed25519 -C "homelab"
```

### 2. Copiar clave al Raspberry Pi
```bash
ssh-copy-id usuario@IP_DEL_RASPBERRY
```

Te pedirá la contraseña una última vez.

### 3. Verificar acceso sin contraseña
```bash
ssh usuario@IP_DEL_RASPBERRY
```

Debe entrar directo sin pedir contraseña.

### 4. Configurar sudo sin contraseña

En el Raspberry Pi:
```bash
sudo visudo
```

Agregar al final (cambia `tu_usuario` por el usuario real):
```
tu_usuario ALL=(ALL) NOPASSWD: ALL
```

### 5. Deshabilitar acceso SSH con contraseña

En el Raspberry Pi:
```bash
sudo nano /etc/ssh/sshd_config
```

Buscar y cambiar estas líneas:
```
PasswordAuthentication no
ChallengeResponseAuthentication no
```

Reiniciar SSH:
```bash
sudo systemctl restart sshd
```

> **⚠️ Importante:** Antes de cerrar la sesión actual, abre otra terminal y verifica que puedes entrar con la clave. Si algo falla, aún tienes la sesión abierta para corregir.

### 6. Verificar que todo funciona
```bash
# Desde tu Mac
ssh usuario@IP_DEL_RASPBERRY      # debe entrar sin contraseña
sudo whoami                        # debe mostrar "root" sin pedir contraseña
```

## Estructura del proyecto
```
homelab-ansible/
├── ansible.cfg
├── inventory/
│   └── inventory.yml
├── playbooks/
│   └── gateway.yml
└── roles/
    ├── wireguard/
    │   ├── defaults/main.yml
    │   ├── handlers/main.yml
    │   ├── tasks/main.yml
    │   └── templates/wg0.conf.j2
    └── dnsmasq/
        ├── defaults/main.yml
        ├── handlers/main.yml
        ├── tasks/main.yml
        ├── templates/
        │   ├── dnsmasq.conf.j2
        │   └── netplan-dns.yaml.j2
        └── README.md
```

## Configuración de Ansible

### 1. Clonar repositorio
```bash
git clone git@github.com:KrlosAren/aren-house.git
cd aren-house/homelab-ansible
```

### 2. Configurar inventario

Editar `inventory/inventory.yml` con las IPs de tus Raspberry Pi:
```yaml
all:
  children:
    gateway:
      hosts:
        rp1-master:
          ansible_host: 192.168.1.84
          ansible_user: rp1-master
          ansible_python_interpreter: /usr/bin/python3
```

### 3. Verificar conexión
```bash
ansible all -m ping
```

Resultado esperado:
```
rp1-master | SUCCESS => {
    "ping": "pong"
}
```

## Uso

### Configurar Gateway completo
```bash
ansible-playbook playbooks/gateway.yml
```

### Modo dry-run (verificar sin aplicar)
```bash
ansible-playbook playbooks/gateway.yml --check
```

## Roles

### wireguard

Instala y configura WireGuard VPN para acceso remoto seguro.

| Variable | Default | Descripción |
|----------|---------|-------------|
| `wireguard_address` | `10.0.1.1/24` | IP del servidor VPN |
| `wireguard_port` | `51820` | Puerto UDP |
| `wireguard_interface` | `wg0` | Nombre de la interfaz |
| `wireguard_peers` | `[]` | Lista de peers |

**Ejemplo de peers:**
```yaml
wireguard_peers:
  - name: mac-xxxx
    public_key: "yhk7pujehS6ZxKSPNWdWveGweO6/uVj6Z0p81H8nnkg="
    allowed_ips: "10.0.1.2/32"
```

### dnsmasq

Instala y configura dnsmasq como servidor DHCP, DNS y TFTP.

| Variable | Default | Descripción |
|----------|---------|-------------|
| `dnsmasq_interface` | `eth0` | Interfaz de red interna |
| `dnsmasq_domain` | `homelab.local` | Dominio local |
| `dnsmasq_network.gateway` | `10.0.0.1` | IP del gateway |
| `dnsmasq_tftp_root` | `/srv/tftp` | Directorio raíz TFTP |
| `dnsmasq_hosts` | `[]` | Lista de hosts con IP fija |

**Ejemplo de hosts:**
```yaml
dnsmasq_hosts:
  - name: rp2
    mac: "2c:cf:67:88:9e:f5"
    ip: "10.0.0.2"
  - name: rp3
    mac: "2c:cf:67:a9:b9:13"
    ip: "10.0.0.3"
  - name: switch
    mac: "ec:75:0c:ff:fc:d6"
    ip: "10.0.0.5"
```

Ver [README del role](roles/dnsmasq/README.md) para más detalles.

## Conectar cliente WireGuard

### macOS

1. Instalar WireGuard desde App Store o:
```bash
brew install wireguard-tools
```

2. Crear configuración `~/homelab.conf`:
```ini
[Interface]
PrivateKey = TU_LLAVE_PRIVADA
Address = 10.0.1.2/24

[Peer]
PublicKey = VIFt08+ZU2nQCnhXAOAMMS+ycH8d6PGLY+hcqZbXhAw=
Endpoint = 192.168.1.84:51820
AllowedIPs = 10.0.0.0/24, 10.0.1.0/24
PersistentKeepalive = 25
```

3. Conectar:
```bash
sudo wg-quick up ~/homelab.conf
```

4. Verificar:
```bash
ping 10.0.1.1   # Gateway VPN
ping 10.0.0.1   # Red interna
ping rp2.homelab.local  # DNS local
```

## Hardware

- 3x Raspberry Pi 5 (Ubuntu Server)
- 1x Switch TP-Link SG105PE (PoE+)
- 1x Adaptador USB-Ethernet
- 1x SSD 250GB (para TFTP/NFS)

## Comandos útiles
```bash
# Probar conexión a todos los hosts
ansible all -m ping

# Ejecutar playbook en modo check (dry-run)
ansible-playbook playbooks/gateway.yml --check

# Ejecutar con más detalle
ansible-playbook playbooks/gateway.yml -v

# Ver estado de WireGuard
sudo wg show

# Conectar/desconectar VPN (Mac)
sudo wg-quick up ~/homelab.conf
sudo wg-quick down ~/homelab.conf

# Ver logs de dnsmasq
sudo tail -f /var/log/dnsmasq.log

# Probar resolución DNS
ping rp2.homelab.local
```

## Troubleshooting

### WireGuard no conecta

1. Verificar servicio: `sudo systemctl status wg-quick@wg0`
2. Verificar puerto: `sudo ss -ulnp | grep 51820`
3. Verificar llaves: la llave **pública** de cada lado va en el `[Peer]` del otro
4. Verificar endpoint: debe ser la IP del gateway (192.168.1.84)

### Ansible no conecta

1. Verificar SSH manual: `ssh usuario@ip`
2. Verificar inventario: `ansible-inventory --list`
3. Verificar ping: `ansible all -m ping -vvv`

### DNS no resuelve nombres locales

1. Verificar que dnsmasq está corriendo: `sudo systemctl status dnsmasq`
2. Verificar netplan: `resolvectl status eth0`
3. Verificar logs: `sudo tail -f /var/log/dnsmasq.log`

### PXE boot no funciona

1. Verificar TFTP: `sudo ss -ulnp | grep 69`
2. Verificar archivos: `ls /srv/tftp/`
3. Verificar NFS exports: `sudo exportfs -v`
4. Verificar EEPROM de la Pi: `sudo rpi-eeprom-config`

## Roadmap

- [x] Gateway/Router con Raspberry Pi
- [x] Segmentación de red
- [x] VPN con WireGuard
- [x] DHCP con IPs fijas (dnsmasq)
- [x] DNS local (.homelab.local)
- [x] TFTP para PXE boot
- [x] NFS para root filesystem
- [x] Netboot de rp2
- [ ] Role NFS en Ansible
- [ ] Role storage (SSD) en Ansible
- [ ] Netboot de rp3
- [ ] Firewall (ufw)
- [ ] Docker en nodos
- [ ] k3s cluster
- [ ] Monitoreo con Prometheus/Grafana

## Decisiones arquitectónicas

Ver [docs/decisions/](../docs/decisions/) para ADRs completos.

| # | Decisión |
|---|----------|
| 001 | [WireGuard sobre OpenVPN](../docs/decisions/001-wireguard-over-openvpn.md) |
| 002 | [Segmentación de red con Raspberry Pi](../docs/decisions/002-network-segmentation.md) |
| 003 | [dnsmasq como DHCP, DNS y TFTP](../docs/decisions/003-dnsmasq-dhcp-dns-tftp.md) |

## Lecciones aprendidas

1. **Siempre usar infraestructura como código** - La memoria falla, el código no
2. **Configurar seguridad SSH antes de todo** - Sin contraseñas, solo claves
3. **Probar en dry-run primero** - `ansible-playbook --check` antes de aplicar
4. **Las llaves públicas van en el peer opuesto** - Fácil de confundir
5. **Segmentación ≠ Seguridad** - Separar redes es el primer paso, firewall es el segundo
6. **Documentar mientras construyes** - No después, se olvidan los detalles
7. **UUID en fstab, no /dev/sdX** - Los nombres de dispositivo pueden cambiar
8. **nofail en fstab** - Para que el sistema bootee aunque el disco no esté
9. **IP forwarding es crítico** - Sin él, el routing entre redes no funciona
10. **Los UIDs difieren entre sistemas** - Cuidado con permisos al copiar filesystems
