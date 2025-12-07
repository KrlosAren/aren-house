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
    └── wireguard/
        ├── defaults/main.yml
        ├── handlers/main.yml
        ├── tasks/main.yml
        └── templates/wg0.conf.j2
```

## Configuración de Ansible

### 1. Clonar repositorio
```bash
git clone <tu-repo>
cd homelab-ansible
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

## Roles

### wireguard

Instala y configura WireGuard VPN.

**Variables:**

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
```

## Redes

| Red | Rango | Propósito |
|-----|-------|-----------|
| LAN Casa | 192.168.1.0/24 | Red principal del modem |
| LAN Homelab | 10.0.0.0/24 | Red interna segmentada |
| VPN | 10.0.1.0/24 | Acceso remoto via túnel |

## Hardware

- 3x Raspberry Pi 5 (Ubuntu Server)
- 1x Switch TP-Link SG105PE (PoE+)
- 1x Adaptador USB-Ethernet

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

### "Permission denied" en SSH

- Verificar que la clave pública está en `~/.ssh/authorized_keys` del RPi
- Verificar permisos: `chmod 600 ~/.ssh/authorized_keys`

### "Sudo requires password"

- Verificar configuración en `/etc/sudoers` con `sudo visudo`

## Roadmap

- [ ] Configurar firewall (ufw) con reglas entre redes
- [ ] Configurar DHCP server en gateway
- [ ] Agregar las otras Raspberry Pi al inventario
- [ ] Instalar Docker en todos los nodos
- [ ] Configurar Pi-hole para DNS interno
- [ ] Monitoreo con Prometheus/Grafana

## Lecciones aprendidas

1. **Siempre usar infraestructura como código** - La memoria falla, el código no
2. **Configurar seguridad SSH antes de todo** - Sin contraseñas, solo claves
3. **Probar en dry-run primero** - `ansible-playbook --check` antes de aplicar
4. **Las llaves públicas van en el peer opuesto** - Fácil de confundir
5. **Segmentación ≠ Seguridad** - Separar redes es el primer paso, firewall es el segundo
6. **Documentar mientras construyes** - No después, se olvidan los detalles