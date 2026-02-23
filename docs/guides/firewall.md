# Guía de Firewall del Homelab

## Estado Actual

El firewall está configurado via Ansible con el playbook `firewall.yml`:

- **Gateway**: UFW que permite todo desde LAN (necesario para netboot), reglas específicas para VPN y WAN
- **Nodos**: UFW con reglas mínimas (SSH desde LAN/VPN, todo desde gateway)

```bash
# Aplicar configuración
ansible-playbook playbooks/firewall.yml

# Ver estado
sudo ufw status verbose
```

Ver [ADR-005: UFW Firewall](../decisions/005-ufw-firewall.md) para decisiones de diseño.

---

## ¿Por qué necesitamos un firewall?

Sin firewall, todos los servicios de tu servidor están expuestos:
```
Internet
    │
    ▼
  Modem ──► rp1-master
            │
            ├── SSH (22) ......... Cualquiera puede intentar login
            ├── DNS (53) ......... Pueden usar tu DNS para ataques
            ├── NFS (2049) ....... Pueden ver tus archivos
            ├── DHCP (67-68) ..... Pueden obtener IPs en tu red
            └── Todo lo demás .... Expuesto
```

Con firewall, controlas qué tráfico permites:
```
Internet
    │
    ▼
  Modem ──► rp1-master
            │
            ├── WireGuard (51820) ✅ Abierto (necesario para VPN)
            ├── SSH (22) ......... ✅ Rate-limited desde WAN
            ├── HTTP/S (80/443) .. ✅ Abierto (servicios web)
            ├── DNS (53) ......... ✅ LAN/VPN/WAN
            ├── NFS (2049) ....... ❌ Solo LAN/VPN
            └── DHCP (67-68) ..... ❌ Solo LAN
```

---

## iptables vs ufw vs nftables

### iptables - El clásico

Es el firewall nativo de Linux desde hace 20+ años. Muy potente pero complejo.
```bash
# Ejemplo: Permitir SSH
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Ejemplo: Bloquear todo lo demás
iptables -A INPUT -j DROP
```

### ufw - Uncomplicated Firewall

Es una capa simplificada sobre iptables. No lo reemplaza, lo hace más fácil de usar.
```bash
# Mismo ejemplo con ufw
ufw allow 22/tcp
ufw default deny incoming
```

### nftables - El sucesor moderno

Reemplaza a iptables en sistemas nuevos. Sintaxis más limpia.
```bash
nft add rule inet filter input tcp dport 22 accept
```

### Comparación

| Aspecto | iptables | ufw | nftables |
|---------|----------|-----|----------|
| Complejidad | Alta | Baja | Media |
| Potencia | Alta | Media | Alta |
| Uso en Ubuntu | Sí | Sí (recomendado) | Sí |
| Curva aprendizaje | Difícil | Fácil | Media |
| Persistencia | Manual | Automática | Manual |

### ¿Por qué elegimos ufw?

1. **Simplicidad** - Comandos fáciles de entender
2. **Ubuntu** - Es el estándar en Ubuntu
3. **Persistencia** - Las reglas sobreviven reinicios automáticamente
4. **Suficiente** - Para un homelab, ufw tiene todo lo necesario

---

## Conceptos de Firewall

### Direcciones del tráfico
```
                    INCOMING                 OUTGOING
                   (entrada)                 (salida)
                       │                        │
Internet ─────────────►│◄─── Tu servidor ──────►│──────────► Internet
                       │                        │
                   ¿Permitir?               ¿Permitir?
```

- **incoming** - Tráfico que ENTRA a tu servidor
- **outgoing** - Tráfico que SALE de tu servidor
- **routed/forward** - Tráfico que PASA a través (NAT)

### Políticas por defecto
```yaml
# Denegar todo lo que entra (seguro)
ufw default deny incoming

# Permitir todo lo que sale (conveniente)
ufw default allow outgoing
```

**¿Por qué deny incoming?** Si permites todo por defecto, cualquier servicio nuevo que instales queda expuesto automáticamente. Es más seguro bloquear todo y abrir solo lo necesario.

**¿Por qué allow outgoing?** Tu servidor necesita descargar actualizaciones (apt), consultar DNS externos, conectar a APIs, sincronizar tiempo (NTP).

---

## Reglas del Gateway (rp1-master)

### Política por defecto
```
Incoming: DENY (bloquear todo lo que entra)
Outgoing: ALLOW (permitir todo lo que sale)
Routed: ALLOW (permitir NAT)
```

### Tabla de reglas

| Puerto | Protocolo | Servicio | Desde | ¿Por qué? |
|--------|-----------|----------|-------|-----------|
| 22/tcp | TCP | SSH | LAN, VPN | Administración remota |
| 22/tcp | TCP | SSH | WAN (limit) | Acceso emergencia, rate-limited contra brute force |
| 51820/udp | UDP | WireGuard | Anywhere | VPN debe ser accesible desde internet |
| 53 | TCP/UDP | DNS | LAN, VPN, WAN | Resolución de nombres (incluyendo `*.k8s.homelab.local` desde WAN) |
| 67-68/udp | UDP | DHCP | LAN (broadcast) | Asignación de IPs a nodos |
| 69/udp | UDP | TFTP | LAN (broadcast) | Netboot de Raspberry Pi |
| 80/tcp | TCP | HTTP | Anywhere | Servicios web (Traefik) |
| 443/tcp | TCP | HTTPS | Anywhere | Servicios web (Traefik) |
| 111/tcp | TCP/UDP | RPC | LAN, VPN | Portmapper para NFS |
| 2049/tcp | TCP/UDP | NFS | LAN, VPN | Filesystem de red |
| 6443/tcp | TCP | k3s API | LAN, Tailscale, WAN | kubectl se conecta aquí |
| 8472/udp | UDP | Flannel VXLAN | LAN | Comunicación entre pods de distintos nodos |
| 9100/tcp | TCP | node_exporter | LAN | Métricas de Prometheus |
| 10250/tcp | TCP | kubelet | LAN | Métricas y logs de pods |
| Todo | - | Netboot | LAN en eth0 | NFS usa puertos dinámicos |

### ¿Por qué "Todo desde LAN"?

NFS usa puertos dinámicos además de 2049 y 111:
```
Cliente NFS:
    ├─► Puerto 111 (portmapper)
    ├─► Puerto 2049 (NFS)
    └─► Puerto 36471 (dinámico) ◄── Bloqueado si no permites todo
```

Por eso necesitamos `allow from 10.0.0.0/24` en la interfaz eth0.

### ¿Por qué DHCP/TFTP por interfaz?

Durante el netboot, rp2 no tiene IP todavía:
```
rp2 bootea:
    └─► DHCP Request desde 0.0.0.0 (no tiene IP)
        │
        ▼
    Firewall: ¿Viene de 10.0.0.0/24? NO (viene de 0.0.0.0)
        │
        ▼
    ❌ BLOQUEADO
```

La solución es permitir por interfaz, no por IP:
```bash
ufw allow in on eth0 to any port 67:68 proto udp
```

### ¿Por qué rate-limit en SSH desde WAN?

`limit` permite máximo 6 conexiones en 30 segundos desde la misma IP:
```
Atacante intenta fuerza bruta:
  Conexión 1 → ✅
  Conexión 2 → ✅
  ...
  Conexión 6 → ✅
  Conexión 7 → ❌ BLOQUEADO (esperar 30 segundos)
```

### Tráfico FORWARD necesario

| Origen | Destino | Propósito |
|--------|---------|-----------|
| 10.0.0.0/24 | Internet | Nodos acceden a internet |
| 10.0.1.0/24 | 10.0.0.0/24 | VPN accede a red interna |
| 10.0.1.0/24 | Internet | VPN accede a internet |
| tailscale0 | * | Tráfico entrante desde Tailscale |
| * | tailscale0 | Tráfico saliente hacia Tailscale |

---

## Reglas de los Nodos (rp2, rp3)

### Tabla de reglas

| Puerto | Servicio | Desde | ¿Por qué? |
|--------|----------|-------|-----------|
| Todo | NFS boot | Gateway (10.0.0.1) | Filesystem completo viene del gateway |
| 22/tcp | SSH | LAN | Administración desde la red local |
| 22/tcp | SSH | VPN | Administración remota via VPN |

### ¿Por qué permitir todo desde gateway?

Los nodos no tienen disco local. Todo viene del gateway:
```
rp2 bootea:
    ├─► TFTP: Descarga kernel
    ├─► NFS: Monta filesystem
    ├─► DNS: Resuelve nombres
    └─► Cualquier conexión del gateway
```

Es más simple permitir todo desde 10.0.0.1 que abrir cada puerto.

---

## Orden de las reglas (MUY IMPORTANTE)
```yaml
# ❌ INCORRECTO - Te bloqueas a ti mismo
1. Política: deny incoming    ◄── SSH bloqueado inmediatamente
2. Allow SSH                  ◄── Nunca llega aquí, ya perdiste acceso
3. Habilitar ufw

# ✅ CORRECTO - Funciona
1. Allow SSH                  ◄── Regla creada (ufw aún deshabilitado)
2. Allow todo desde gateway   ◄── Regla creada
3. Política: deny incoming    ◄── Configurada pero no aplicada
4. Habilitar ufw              ◄── Ahora aplica todo junto
```

En Ansible, esto significa crear las reglas de `allow` ANTES de configurar las políticas `deny`.

---

## Diagrama de tráfico completo
```
                         INTERNET
                             │
                             ▼
                    ┌─────────────────┐
                    │   Modem/Router  │
                    │  192.168.1.89.1  │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
         WireGuard        SSH           HTTP/S
         51820/udp      22/tcp         80,443/tcp
            ✅         ✅ limit           ✅
              │              │              │
              └──────────────┼──────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   rp1-master    │
                    │    GATEWAY      │
                    │    10.0.0.1     │
                    │                 │
                    │  ┌───────────┐  │
                    │  │  FIREWALL │  │
                    │  │   (ufw)   │  │
                    │  └───────────┘  │
                    │                 │
                    │  HTTP/S → DNAT  │
                    │  → 10.0.0.50   │
                    │  (MetalLB/     │
                    │   Traefik k3s) │
                    └────────┬────────┘
                             │
                       eth0 (LAN)
                    Todo permitido
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
        ┌──────────┐                  ┌──────────┐
        │   rp2    │                  │   rp3    │
        │ 10.0.0.2 │                  │ 10.0.0.3 │
        │          │                  │          │
        │ FIREWALL │                  │ FIREWALL │
        │ GW: ✅   │                  │ GW: ✅   │
        │ SSH: ✅  │                  │ SSH: ✅  │
        └──────────┘                  └──────────┘


                     Tailscale VPN
                     100.x.x.x
                    (tu Mac remoto)
                          │
                          ▼
                    Acceso a:
                    - SSH (todos)
                    - DNS
                    - NFS
                    - kubectl (6443)

              Clientes WAN (192.168.1.x)
                          │
                          ▼
                    Acceso a:
                    - DNS (53) → resolver *.k8s.homelab.local
                    - HTTP/S → DNAT → MetalLB (Traefik k3s)
                    - k3s API (6443)
```

---

## Configuración con UFW

### Habilitar forwarding en UFW

Editar `/etc/ufw/sysctl.conf`:
```bash
net/ipv4/ip_forward=1
```

### Configurar NAT en UFW

Editar `/etc/ufw/before.rules`, agregar al inicio (antes de `*filter`):
```bash
# NAT para red interna + DNAT para servicios k8s
*nat
:PREROUTING ACCEPT [0:0]
:POSTROUTING ACCEPT [0:0]
# Nodos acceden a internet
-A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
# WAN HTTP/HTTPS → MetalLB (Traefik k3s)
-A PREROUTING -i enx00e04c683da2 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.50:80
-A PREROUTING -i enx00e04c683da2 -p tcp --dport 443 -j DNAT --to-destination 10.0.0.50:443
COMMIT
```

### DNAT: Acceso WAN a servicios k8s

Las reglas DNAT redirigen tráfico HTTP/HTTPS que llega por la interfaz WAN hacia MetalLB (Traefik k3s):

```
Cliente WAN (192.168.1.x)
       │
       │ http://app.k8s.homelab.local
       ▼
  enx00e04c683da2 (192.168.1.89)
       │
       │ DNAT: --dport 80 → 10.0.0.50:80
       │ DNAT: --dport 443 → 10.0.0.50:443
       ▼
  MetalLB → Traefik k3s → Ingress → Service → Pod
```

Esto permite que equipos en la red del modem (192.168.1.0/24) accedan a servicios k8s sin necesidad de Tailscale. El playbook `firewall.yml` inyecta estas reglas automáticamente en `/etc/ufw/before.rules`.

### Reglas básicas
```bash
# Política por defecto
sudo ufw default deny incoming
sudo ufw default allow outgoing

# SSH (desde redes internas y VPN)
sudo ufw allow from 10.0.0.0/24 to any port 22 proto tcp
sudo ufw allow from 10.0.1.0/24 to any port 22 proto tcp

# DNS
sudo ufw allow from 10.0.0.0/24 to any port 53

# DHCP (broadcast por interfaz)
sudo ufw allow in on eth0 to any port 67:68 proto udp

# TFTP
sudo ufw allow from 10.0.0.0/24 to any port 69 proto udp

# NFS
sudo ufw allow from 10.0.0.0/24 to any port 111
sudo ufw allow from 10.0.0.0/24 to any port 2049

# WireGuard (desde cualquier lugar en WAN)
sudo ufw allow 51820/udp

# Habilitar
sudo ufw enable
```

## Configuración con iptables (Manual)

Si prefieres iptables directamente sin UFW:

### Script de configuración

Crear `/etc/iptables/rules.sh`:
```bash
#!/bin/bash

# Limpiar reglas existentes
iptables -F
iptables -X
iptables -t nat -F

# Políticas por defecto
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Permitir loopback
iptables -A INPUT -i lo -j ACCEPT

# Permitir conexiones establecidas
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT

# SSH desde redes internas
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -s 10.0.1.0/24 -p tcp --dport 22 -j ACCEPT

# DNS desde red interna
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 53 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 53 -j ACCEPT

# DHCP desde red interna
iptables -A INPUT -i eth0 -p udp --dport 67:68 -j ACCEPT

# TFTP desde red interna
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 69 -j ACCEPT

# NFS desde red interna
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 111 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 111 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 2049 -j ACCEPT
iptables -A INPUT -s 10.0.0.0/24 -p udp --dport 2049 -j ACCEPT

# WireGuard desde WAN
iptables -A INPUT -p udp --dport 51820 -j ACCEPT

# FORWARD: Red interna a internet
iptables -A FORWARD -i eth0 -o enx00e04c683da2 -j ACCEPT

# FORWARD: VPN a red interna
iptables -A FORWARD -i wg0 -o eth0 -j ACCEPT
iptables -A FORWARD -i eth0 -o wg0 -j ACCEPT

# FORWARD: VPN a internet
iptables -A FORWARD -i wg0 -o enx00e04c683da2 -j ACCEPT

# NAT para salida a internet
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
iptables -t nat -A POSTROUTING -s 10.0.1.0/24 -o enx00e04c683da2 -j MASQUERADE
```

### Hacer reglas persistentes
```bash
sudo apt install iptables-persistent
sudo netfilter-persistent save
# Las reglas se guardan en /etc/iptables/rules.v4 y rules.v6
```

---

## Notas sobre k3s y Tailscale

### k3s

El playbook `k3s.yml` agrega automáticamente reglas iptables para FORWARD de Tailscale y las persiste con `iptables-persistent`. Los puertos críticos de k3s son:

- **6443/TCP** - API Server (kubectl se conecta aquí)
- **8472/UDP** - Flannel VXLAN (comunicación entre pods de distintos nodos)
- **10250/TCP** - kubelet (métricas y logs de pods)

El playbook `firewall.yml` permite todo el tráfico desde la LAN en eth0 (`ufw allow in on eth0 from 10.0.0.0/24`), lo que cubre estos puertos para comunicación entre nodos.

### Tailscale

Tailscale maneja su propio tunnel y no necesita puertos explícitos en UFW. Sin embargo, el playbook `k3s.yml` agrega reglas FORWARD para la interfaz `tailscale0` para permitir que el tráfico de kubectl via Tailscale llegue al API Server.

---

## Comandos útiles

### Ver estado
```bash
# Estado general
sudo ufw status verbose

# Reglas numeradas (útil para eliminar)
sudo ufw status numbered

# Ver reglas en formato iptables
sudo iptables -L -n -v

# Ver reglas NAT
sudo iptables -t nat -L -n -v
```

### Administrar reglas
```bash
# Agregar regla
sudo ufw allow from 10.0.0.0/24 to any port 22

# Eliminar regla por número
sudo ufw status numbered
sudo ufw delete 5

# Eliminar regla específica
sudo ufw delete allow 80/tcp
```

### Administrar firewall
```bash
# Habilitar
sudo ufw enable

# Deshabilitar
sudo ufw disable

# Resetear todo
sudo ufw --force reset

# Recargar reglas
sudo ufw reload
```

### Logs
```bash
# Ver logs de bloqueos en tiempo real
sudo tail -f /var/log/ufw.log

# Buscar bloqueos de una IP específica
sudo grep "10.0.0.2" /var/log/ufw.log

# Habilitar logging de paquetes rechazados (iptables)
iptables -A INPUT -j LOG --log-prefix "IPT-INPUT-DROP: " --log-level 4
iptables -A FORWARD -j LOG --log-prefix "IPT-FORWARD-DROP: " --log-level 4
```

### Probar conectividad
```bash
# Desde un nodo (10.0.0.x)
ping 8.8.8.8           # Internet
ping 10.0.0.1          # Gateway
nslookup google.com    # DNS

# Desde VPN (10.0.1.x)
ping 10.0.0.2          # Nodo interno
ssh admin@10.0.0.2     # SSH a nodo
```

---

## Recuperación de emergencia

### Si pierdes acceso a los nodos (NFS boot)

Los nodos bootean por NFS, así que puedes editar sus archivos desde el gateway:
```bash
# Desde rp1-master, deshabilitar ufw en rp2
sudo sed -i 's/ENABLED=yes/ENABLED=no/' /srv/nfs/rp2/etc/ufw/ufw.conf

# Verificar
cat /srv/nfs/rp2/etc/ufw/ufw.conf | grep ENABLED

# Reiniciar el nodo (desconectar/conectar alimentación)
```

### Si pierdes acceso al gateway

Si tienes acceso físico (monitor + teclado):
```bash
sudo ufw disable
# O resetear
sudo ufw --force reset
```

Si no tienes acceso físico, necesitarás acceso por consola serial o reinstalar.

---

## Troubleshooting

### Netboot falla después de habilitar firewall

**Síntoma:** rp2/rp3 no pueden bootear
**Causa:** DHCP o TFTP bloqueado
**Solución:**
```bash
sudo tail -f /var/log/ufw.log
sudo ufw allow in on eth0 to any port 67:68 proto udp
sudo ufw allow in on eth0 to any port 69 proto udp
sudo ufw allow in on eth0 from 10.0.0.0/24
```

### No puedo conectar por SSH desde VPN

**Síntoma:** SSH timeout desde Mac con VPN
**Causa:** Tu IP de VPN no está en las reglas
**Verificar:**
```bash
ifconfig utun8 | grep inet        # Tu IP de VPN (Mac)
sudo ufw status | grep 10.0.1     # Reglas en el servidor
```
**Solución:**
```bash
sudo ufw allow from 10.0.1.0/24 to any port 22
```

### NFS no funciona pero ping sí

**Síntoma:** Puedes hacer ping pero NFS timeout
**Causa:** NFS usa puertos dinámicos que están bloqueados
**Solución:**
```bash
sudo ufw allow in on eth0 from 10.0.0.0/24
```

### Los nodos no tienen internet después de habilitar firewall

1. Verificar regla FORWARD: `sudo iptables -L FORWARD -n -v`
2. Verificar NAT: `sudo iptables -t nat -L POSTROUTING -n -v`
3. Verificar IP forwarding: `cat /proc/sys/net/ipv4/ip_forward` (debe ser 1)

### WireGuard no conecta

1. Verificar puerto: `sudo iptables -L INPUT -n | grep 51820`
2. Verificar servicio: `sudo wg show`

### Los nodos no bootean (TFTP/NFS)

1. Verificar puertos: `sudo iptables -L INPUT -n | grep -E "69|2049|111"`
2. Temporalmente deshabilitar firewall: `sudo ufw disable`

---

## Próximos Pasos

1. ~~Crear rol Ansible para firewall~~ (completado: `playbooks/firewall.yml`)
2. ~~Agregar rate limiting para protección contra fuerza bruta~~ (completado: `ufw limit` en SSH)
3. ~~Revisar reglas HTTP/HTTPS para servicios web~~ (completado: Traefik via MetalLB)
4. Considerar fail2ban para SSH
5. Agregar logging de paquetes rechazados
