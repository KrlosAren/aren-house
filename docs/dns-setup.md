# Configuración DNS del Homelab

## Arquitectura
```
                    Internet
                        │
                        ▼
              ┌─────────────────┐
              │  Modem/Router   │
              │  192.168.1.89.1  │
              │  (DNS público)  │
              └────────┬────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   rp1-master    │
              │    10.0.0.1     │
              │  dnsmasq (DNS)  │◄── DNS local para .homelab.local
              └────────┬────────┘
                       │
           ┌───────────┴───────────┐
           ▼                       ▼
    ┌─────────────┐         ┌─────────────┐
    │  rp2-node   │         │  rp3-node   │
    │  10.0.0.2   │         │  10.0.0.3   │
    └─────────────┘         └─────────────┘
```

## Servidor DNS (dnsmasq en rp1-master)

### Ubicación de configuración
```
/etc/dnsmasq.conf
```

### Interfaces

dnsmasq escucha en tres interfaces:

```ini
# LAN - DHCP + DNS
interface=eth0

# WAN - solo DNS (para resolver *.k8s.homelab.local desde la red del modem)
interface=enx00e04c683da2
no-dhcp-interface=enx00e04c683da2

# Tailscale - solo DNS (para que el Mac y otros dispositivos Tailscale resuelvan)
interface=tailscale0
no-dhcp-interface=tailscale0

# También escucha en localhost
listen-address=127.0.0.1

# bind-dynamic permite escuchar en múltiples interfaces sin conflictos
bind-dynamic
```

**¿Por qué dnsmasq en WAN?** Los clientes en la red del modem (192.168.1.0/24) necesitan resolver `*.k8s.homelab.local` para acceder a servicios k8s via DNAT. Sin esto, solo los clientes en la LAN (10.0.0.0/24) pueden resolver esos nombres.

**¿Por qué dnsmasq en Tailscale?** Los dispositivos que se conectan via Tailscale (ej. Mac en otra red) consultan dnsmasq por su IP Tailscale (`100.x.x.x`). Sin `interface=tailscale0`, esas queries llegarían a rp1 pero dnsmasq las ignoraría.

### Tipos de registros

#### Para clientes DHCP (rp2, rp3)
```ini
dhcp-host=2c:cf:67:88:9e:f5,rp2,10.0.0.2
dhcp-host=2c:cf:67:a9:b9:13,rp3,10.0.0.3
```

Esto hace dos cosas:
1. Asigna IP fija por DHCP
2. Crea registro DNS automáticamente

#### Para hosts estáticos (rp1)
```ini
host-record=rp1,rp1.homelab.local,10.0.0.1
```

Solo crea el registro DNS (el host ya tiene IP estática).

### Dominio local
```ini
domain=homelab.local
local=/homelab.local/
```

## Acceso a apps según dispositivo

| Dispositivo | Cómo accede | Configuración extra |
|-------------|-------------|---------------------|
| LAN física (10.0.0.0/24) | dnsmasq via DHCP | Ninguna — funciona automáticamente |
| Mac via Tailscale | dnsmasq via Tailscale IP de rp1 | `/etc/resolver/` (ver abajo) |
| Otro dispositivo via Tailscale | igual que Mac | mismos archivos `/etc/resolver/` |
| Internet (sin Tailscale) | No disponible | IP WAN dinámica + CGNAT |

## Cliente DNS (macOS con Tailscale)

### El flujo

Cuando el Mac consulta `grafana.k8s.homelab.local`, macOS usa `/etc/resolver/k8s.homelab.local` para saber qué nameserver usar. Ese nameserver es dnsmasq en rp1, accesible via su IP de Tailscale.

```
Mac → /etc/resolver/k8s.homelab.local → dig @<IP_Tailscale_rp1>
    → dnsmasq responde: 192.168.1.89
    → DNAT :80 → MetalLB 10.0.0.50 → Traefik → Pod
```

### Configurar (primera vez)

```bash
# Obtener IP Tailscale actual de rp1
tailscale status | grep rp1-master

# Crear resolvers (reemplazar con la IP obtenida)
sudo bash -c '
  echo "nameserver <IP_TAILSCALE_RP1>" > /etc/resolver/homelab.local
  echo "nameserver <IP_TAILSCALE_RP1>" > /etc/resolver/k8s.homelab.local
'
```

**Nota**: La IP Tailscale de rp1 es estable entre reboots pero puede cambiar si se reinstala Tailscale. Verificar con `tailscale status` si el DNS deja de funcionar.

### Si el DNS deja de resolver

```bash
# 1. Verificar IP actual de rp1 en Tailscale
tailscale status | grep rp1-master

# 2. Actualizar los archivos si la IP cambió
sudo bash -c 'echo "nameserver <NUEVA_IP>" > /etc/resolver/homelab.local'
sudo bash -c 'echo "nameserver <NUEVA_IP>" > /etc/resolver/k8s.homelab.local'

# 3. Verificar que dnsmasq responde
dig @<IP_TAILSCALE_RP1> grafana.k8s.homelab.local
```

### Alternativa recomendada: Split DNS en Tailscale Admin

En vez de configurar `/etc/resolver/` manualmente en cada dispositivo, se puede centralizar en la consola de Tailscale y pushear a todos los dispositivos automáticamente:

1. Ir a [Tailscale Admin → DNS](https://login.tailscale.com/admin/dns)
2. **Add nameserver** → Custom → IP Tailscale de rp1
3. Marcar **Restrict to domain**: `homelab.local`
4. Repetir para `k8s.homelab.local`

### Verificar
```bash
# Consultar DNS a través de Tailscale
dig @<IP_TAILSCALE_RP1> grafana.k8s.homelab.local

# Ver resolvers activos en macOS
scutil --dns | grep -A3 homelab

# Probar curl completo
curl http://grafana.k8s.homelab.local/
```

## Comandos útiles

### En el servidor (rp1-master)
```bash
# Ver logs de DNS
sudo tail -f /var/log/dnsmasq.log

# Reiniciar dnsmasq
sudo systemctl restart dnsmasq

# Ver leases DHCP activos
cat /var/lib/misc/dnsmasq.leases

# Probar resolución local
dig @localhost rp2.homelab.local
```

### En clientes
```bash
# Consultar DNS
nslookup rp2.homelab.local 10.0.0.1
dig @10.0.0.1 rp2.homelab.local

# Ver qué DNS usa tu sistema
cat /etc/resolv.conf        # Linux
scutil --dns                 # macOS
```

## Agregar nuevos hosts

### Si el host usa DHCP (nodos worker)

Agregar en `roles/dnsmasq/defaults/main.yml`:
```yaml
dnsmasq_hosts:
  - mac: "aa:bb:cc:dd:ee:ff"
    name: "rp4"
    ip: "10.0.0.4"
```

### Si el host tiene IP estática

Agregar en el template `roles/dnsmasq/templates/dnsmasq.conf.j2`:
```ini
host-record=nombre,nombre.homelab.local,IP
```

## DNS para Kubernetes (k3s)

### Dominios

El homelab usa dos dominios DNS con propósitos diferentes:

```
*.homelab.local      → 10.0.0.1      (servicios Docker en rp1-master, via Traefik Docker)
*.k8s.homelab.local  → 192.168.1.89  (servicios k8s, via DNAT → MetalLB 10.0.0.50)
```

**¿Por qué la IP WAN (192.168.1.89) y no la de MetalLB (10.0.0.50)?**

Los clientes en la red WAN (192.168.1.0/24) no tienen ruta directa a 10.0.0.50. El tráfico llega a la IP WAN del gateway, donde reglas DNAT en el firewall redirigen HTTP/HTTPS a MetalLB:

```
Cliente WAN → 192.168.1.89:80 → DNAT → 10.0.0.50:80 (MetalLB/Traefik k3s)
```

Para clientes en la LAN (10.0.0.0/24) o Tailscale, la resolución también funciona porque el gateway tiene la IP 192.168.1.89 en su interfaz WAN.

### Cómo se configura

La entrada DNS de k8s se define en el role de dnsmasq (`roles/dnsmasq/templates/dnsmasq.conf.j2`):

```ini
address=/.k8s.homelab.local/192.168.1.89
```

Las reglas DNAT en el firewall (`playbooks/firewall.yml`) redirigen el tráfico:

```
# En /etc/ufw/before.rules
*nat
:PREROUTING ACCEPT [0:0]
-A PREROUTING -i enx00e04c683da2 -p tcp --dport 80 -j DNAT --to-destination 10.0.0.50:80
-A PREROUTING -i enx00e04c683da2 -p tcp --dport 443 -j DNAT --to-destination 10.0.0.50:443
COMMIT
```

### CoreDNS (DNS interno del cluster)

k3s incluye CoreDNS para resolución DNS dentro del cluster. Los pods usan CoreDNS para resolver:
- Nombres de Services: `mi-svc.mi-namespace.svc.cluster.local`
- Nombres externos: se reenvían al DNS del nodo (dnsmasq)

CoreDNS no afecta la resolución DNS de clientes fuera del cluster.

### macOS

Si usas Tailscale, necesitas un resolver adicional para el dominio k8s:

```bash
sudo bash -c 'echo "nameserver 10.0.0.1" > /etc/resolver/k8s.homelab.local'
```

## DNS con Tailscale

### MagicDNS

Tailscale incluye **MagicDNS**, que asigna nombres DNS automáticos a los dispositivos de la red Tailscale. Si lo habilitas en la consola de Tailscale:

- `rp1-master` sería accesible como `rp1-master.<tailnet-name>.ts.net`
- No requiere configuración en dnsmasq

Para habilitarlo: [Tailscale Admin → DNS](https://login.tailscale.com/admin/dns)

### Limitación actual

MagicDNS solo resuelve dispositivos con Tailscale instalado. Los nodos rp2 y rp3 no tienen Tailscale (acceden via subnet routing), por lo que no son resolubles por MagicDNS. Para acceder a ellos por nombre, se sigue usando dnsmasq + el resolver local en macOS.

## Problema: systemd-resolved + Tailscale MagicDNS

### El problema

Los nodos worker (rp2, rp3) usan `systemd-resolved` por defecto en Ubuntu, que configura `/etc/resolv.conf` con `nameserver 127.0.0.53`. Cuando Tailscale está activo, `systemd-resolved` usa MagicDNS como upstream, que no conoce los dominios `.homelab.local` ni `.k8s.homelab.local`.

```
# Flujo ROTO:
Pod/containerd → 127.0.0.53 → systemd-resolved → Tailscale MagicDNS → ??? (.k8s.homelab.local)
                                                                         ↓
                                                                    No resuelve
```

### La solución

Deshabilitar `systemd-resolved` y crear un `/etc/resolv.conf` estático apuntando a dnsmasq:

```bash
# En cada nodo worker (rp2, rp3):
sudo systemctl stop systemd-resolved
sudo systemctl disable systemd-resolved
sudo rm /etc/resolv.conf
sudo bash -c 'cat > /etc/resolv.conf << EOF
nameserver 10.0.0.1
search homelab.local
EOF'
```

```
# Flujo CORRECTO:
Pod/containerd → 10.0.0.1 → dnsmasq → resuelve .homelab.local / .k8s.homelab.local
                                     → reenvía al upstream para dominios externos
```

### Playbook dns-client.yml

Esta configuración está automatizada con el playbook `dns-client.yml`:

```bash
cd homelab-ansible
ansible-playbook playbooks/dns-client.yml
```

El playbook:
1. Detecta si `systemd-resolved` está activo
2. Detiene y deshabilita `systemd-resolved`
3. Elimina el symlink `/etc/resolv.conf`
4. Crea `/etc/resolv.conf` estático con `nameserver 10.0.0.1`
5. Verifica resolución de `*.k8s.homelab.local`

## Configuración containerd: registries.yaml

### Por qué es necesario

containerd (el runtime de k3s) necesita saber que el registry local usa HTTP, no HTTPS. Además, containerd no usa el DNS del sistema (dnsmasq) por defecto, así que necesita `/etc/hosts` para resolver el nombre del registry.

### `/etc/rancher/k3s/registries.yaml`

```yaml
mirrors:
  registry.k8s.homelab.local:
    endpoint:
      - "http://registry.k8s.homelab.local"
  docker.io:
    endpoint:
      - "https://registry-1.docker.io"
```

### `/etc/hosts` en cada nodo

```
10.0.0.50 registry.k8s.homelab.local
```

### Flujo de pull de imagen

```
1. Pod spec: image: registry.k8s.homelab.local/test-app:latest
2. containerd lee registries.yaml → usa HTTP (no HTTPS)
3. Resuelve "registry.k8s.homelab.local" via /etc/hosts → 10.0.0.50
4. HTTP GET → MetalLB (10.0.0.50) → Traefik k3s → Ingress → registry Pod
5. Imagen descargada y cacheada en el nodo
```

### Automatización

```bash
cd homelab-ansible
ansible-playbook playbooks/registry.yml
```

El playbook configura `registries.yaml` y reinicia k3s/k3s-agent en cada nodo.

## Troubleshooting

| Problema | Causa | Solución |
|----------|-------|----------|
| `nslookup` funciona pero `ssh nombre` no | macOS no usa el DNS correcto | Crear `/etc/resolver/homelab.local` |
| Nombre no resuelve | dnsmasq no tiene el registro | Verificar `/etc/dnsmasq.conf` |
| DNS lento | Servidor upstream no responde | Verificar `dnsmasq_dns_servers` |
| `*.k8s.homelab.local` no resuelve | Entrada faltante en dnsmasq | Verificar `address=/.k8s.homelab.local/192.168.1.89` en dnsmasq.conf |
| `*.k8s.homelab.local` no resuelve en nodos | systemd-resolved + Tailscale | Deshabilitar systemd-resolved, apuntar a 10.0.0.1 |
| DNS no resuelve desde Mac via Tailscale | IP Tailscale de rp1 cambió en `/etc/resolver/` | Actualizar con `tailscale status \| grep rp1-master` y reescribir `/etc/resolver/` |
| DNS no resuelve desde Mac via Tailscale | dnsmasq no escucha en `tailscale0` | Agregar `interface=tailscale0` al template y aplicar `gateway.yml` |
| `*.k8s.homelab.local` resuelve pero no conecta desde WAN | DNAT no configurado | Verificar reglas NAT en `/etc/ufw/before.rules` y ejecutar `playbooks/firewall.yml` |
| Pod no resuelve DNS externo | CoreDNS no puede reenviar | Verificar que dnsmasq acepta queries desde la red de pods (10.42.0.0/16) |
| Pull de imagen falla con `server gave HTTP response to HTTPS` | containerd intenta HTTPS | Configurar mirror HTTP en `registries.yaml`, reiniciar k3s |
