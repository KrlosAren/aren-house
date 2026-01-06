# UFW (Uncomplicated Firewall)

UFW es una interfaz simplificada para gestionar iptables en Linux.

## Qué es UFW

UFW no reemplaza iptables, sino que genera reglas iptables automáticamente con una sintaxis más simple. Es el firewall por defecto en Ubuntu.

```
┌─────────────────────────────────────┐
│           Comandos UFW              │
│   ufw allow 22, ufw deny 80, etc.   │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│         Reglas UFW                  │
│   /etc/ufw/user.rules               │
│   /etc/ufw/before.rules             │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│           iptables                  │
│   Reglas reales del kernel          │
└─────────────────────────────────────┘
```

## Conceptos Básicos

### Políticas por Defecto

Definen qué hacer con tráfico que no coincide con ninguna regla:

```bash
# Denegar entrada por defecto (seguro)
sudo ufw default deny incoming

# Permitir salida por defecto
sudo ufw default allow outgoing
```

### Direcciones

- **incoming**: Tráfico que entra al host
- **outgoing**: Tráfico que sale del host
- **routed**: Tráfico que pasa a través (forwarding)

### Estados

- **enabled**: Firewall activo, bloqueando según reglas
- **disabled**: Firewall inactivo, todo pasa
- **reloaded**: Recargar reglas sin desactivar

## Sintaxis de Reglas

### Formato Básico

```bash
ufw [allow|deny|limit] [from IP] [to any] [port PORT] [proto PROTOCOL]
```

### Ejemplos

```bash
# Permitir SSH desde cualquier lugar
sudo ufw allow 22/tcp

# Permitir SSH solo desde red local
sudo ufw allow from 10.0.0.0/24 to any port 22 proto tcp

# Permitir DNS (TCP y UDP)
sudo ufw allow 53

# Limitar SSH (bloquea IPs con muchos intentos)
sudo ufw limit 22/tcp

# Denegar puerto específico
sudo ufw deny 3306/tcp

# Permitir rango de puertos
sudo ufw allow 6000:6007/tcp

# Regla con comentario
sudo ufw allow from 10.0.0.0/24 to any port 22 comment "SSH desde LAN"
```

## Rate Limiting

`ufw limit` bloquea IPs que hacen más de 6 conexiones en 30 segundos:

```bash
sudo ufw limit 22/tcp
```

Útil para proteger SSH contra ataques de fuerza bruta.

## Forwarding (NAT)

UFW puede manejar forwarding para routers/gateways:

### 1. Habilitar forwarding

Editar `/etc/default/ufw`:
```bash
DEFAULT_FORWARD_POLICY="ACCEPT"
```

### 2. Habilitar en sysctl

Editar `/etc/ufw/sysctl.conf`:
```bash
net/ipv4/ip_forward=1
```

### 3. Agregar regla NAT

Editar `/etc/ufw/before.rules`, agregar antes de `*filter`:
```bash
*nat
:POSTROUTING ACCEPT [0:0]
-A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
COMMIT
```

## Archivos de Configuración

| Archivo | Propósito |
|---------|-----------|
| `/etc/default/ufw` | Configuración global (forwarding, IPv6) |
| `/etc/ufw/ufw.conf` | Estado enabled/disabled |
| `/etc/ufw/before.rules` | Reglas que se aplican primero (NAT, ICMP) |
| `/etc/ufw/user.rules` | Reglas del usuario (generadas por comandos) |
| `/etc/ufw/after.rules` | Reglas que se aplican al final |
| `/etc/ufw/sysctl.conf` | Parámetros de red (ip_forward) |

## Comandos Útiles

```bash
# Ver estado y reglas
sudo ufw status verbose

# Ver reglas con números (para eliminar)
sudo ufw status numbered

# Eliminar regla por número
sudo ufw delete 3

# Eliminar regla por definición
sudo ufw delete allow 22/tcp

# Resetear todas las reglas
sudo ufw reset

# Habilitar firewall
sudo ufw enable

# Deshabilitar firewall
sudo ufw disable

# Recargar reglas
sudo ufw reload
```

## UFW vs iptables

| Aspecto | UFW | iptables |
|---------|-----|----------|
| Sintaxis | Simple | Compleja |
| Curva de aprendizaje | Baja | Alta |
| Comentarios | Sí | No nativo |
| Rate limiting | `limit` | Requiere módulos |
| Persistencia | Automática | Requiere iptables-persistent |
| Control total | No | Sí |

## Integración con Ansible

Ansible tiene módulo nativo para UFW:

```yaml
- name: Permitir SSH desde LAN
  ufw:
    rule: allow
    port: "22"
    proto: tcp
    from_ip: "10.0.0.0/24"
    comment: "SSH desde LAN"

- name: Habilitar UFW
  ufw:
    state: enabled
```

## Troubleshooting

### Ver qué está bloqueando

```bash
# Habilitar logging
sudo ufw logging on

# Ver logs
sudo tail -f /var/log/ufw.log
```

### Reglas no se aplican

```bash
# Verificar que UFW está habilitado
sudo ufw status

# Recargar si es necesario
sudo ufw reload
```

### Bloqueado de SSH

Si te bloqueas el acceso SSH:
1. Acceso físico o consola
2. `sudo ufw disable` o `sudo ufw allow 22`

### Conflicto con Docker

Docker modifica iptables directamente, puede saltarse reglas UFW:
- Usar `DOCKER_OPTS="--iptables=false"` si es necesario control total

## Referencias

- [UFW Community Help](https://help.ubuntu.com/community/UFW)
- [UFW Man Page](https://manpages.ubuntu.com/manpages/jammy/man8/ufw.8.html)
- [iptables basics](./iptables-basics.md)
