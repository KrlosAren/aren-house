# Guía de Firewall del Homelab

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
            ├── DNS (53) ......... ❌ Solo LAN/VPN
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

**¿Por qué deny incoming?**

Si permites todo por defecto, cualquier servicio nuevo que instales queda expuesto automáticamente. Es más seguro bloquear todo y abrir solo lo necesario.

**¿Por qué allow outgoing?**

Tu servidor necesita:
- Descargar actualizaciones (apt)
- Consultar DNS externos
- Conectar a APIs
- Sincronizar tiempo (NTP)

---

## Reglas del Gateway (rp1-master)

### Política por defecto
```
Incoming: DENY (bloquear todo lo que entra)
Outgoing: ALLOW (permitir todo lo que sale)
Routed: ALLOW (permitir NAT)
```

### Tabla de reglas

| Puerto | Servicio | Desde | ¿Por qué? |
|--------|----------|-------|-----------|
| 22/tcp | SSH | LAN, VPN | Administración remota |
| 22/tcp | SSH | WAN (limit) | Acceso emergencia, rate-limited contra brute force |
| 51820/udp | WireGuard | Anywhere | VPN debe ser accesible desde internet |
| 53 | DNS | LAN, VPN | Resolución de nombres local |
| 67-68/udp | DHCP | LAN (broadcast) | Asignación de IPs a nodos |
| 69/udp | TFTP | LAN (broadcast) | Netboot de Raspberry Pi |
| 111/tcp | RPC | LAN, VPN | Portmapper para NFS |
| 2049/tcp | NFS | LAN, VPN | Filesystem de red |
| 80/tcp | HTTP | Anywhere | Servicios web (Traefik) |
| 443/tcp | HTTPS | Anywhere | Servicios web (Traefik) |
| 6443/tcp | k3s API | LAN, Tailscale | kubectl se conecta aquí |
| 8472/udp | Flannel VXLAN | LAN | Comunicación entre pods de distintos nodos |
| 10250/tcp | kubelet | LAN | Métricas y logs de pods |
| Todo | Netboot | LAN en eth0 | NFS usa puertos dinámicos |

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
                    │  192.168.100.1  │
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
```

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
# Deshabilitar firewall
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
# Verificar logs
sudo tail -f /var/log/ufw.log

# Agregar reglas por interfaz
sudo ufw allow in on eth0 to any port 67:68 proto udp
sudo ufw allow in on eth0 to any port 69 proto udp
sudo ufw allow in on eth0 from 10.0.0.0/24
```

### No puedo conectar por SSH desde VPN

**Síntoma:** SSH timeout desde Mac con VPN

**Causa:** Tu IP de VPN no está en las reglas

**Verificar:**
```bash
# En tu Mac, ver tu IP de VPN
ifconfig utun8 | grep inet

# En el servidor, verificar reglas
sudo ufw status | grep 10.0.1
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
# Permitir todo desde LAN
sudo ufw allow in on eth0 from 10.0.0.0/24
```
