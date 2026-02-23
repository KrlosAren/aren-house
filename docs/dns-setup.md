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

dnsmasq escucha en dos interfaces:

```ini
# LAN - DHCP + DNS
interface=eth0

# WAN - solo DNS (para resolver *.k8s.homelab.local desde la red del modem)
interface=enx00e04c683da2
no-dhcp-interface=enx00e04c683da2

# También escucha en localhost
listen-address=127.0.0.1

# bind-dynamic permite escuchar en múltiples interfaces sin conflictos
bind-dynamic
```

**¿Por qué dnsmasq en WAN?** Los clientes en la red del modem (192.168.1.0/24) necesitan resolver `*.k8s.homelab.local` para acceder a servicios k8s via DNAT. Sin esto, solo los clientes en la LAN (10.0.0.0/24) o VPN pueden resolver esos nombres.

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

## Cliente DNS (macOS con VPN)

### Problema

macOS usa el DNS del modem (192.168.1.89.1) por defecto, que no conoce `.homelab.local`.

### Solución

Crear un resolver específico para el dominio:
```bash
sudo mkdir -p /etc/resolver
sudo bash -c 'echo "nameserver 10.0.0.1" > /etc/resolver/homelab.local'
```

### Verificar
```bash
# Consultar DNS específico
nslookup rp1.homelab.local 10.0.0.1

# Ver configuración DNS de macOS
scutil --dns

# Probar conexión
ssh admin@rp1.homelab.local
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

## Troubleshooting

| Problema | Causa | Solución |
|----------|-------|----------|
| `nslookup` funciona pero `ssh nombre` no | macOS no usa el DNS correcto | Crear `/etc/resolver/homelab.local` |
| Nombre no resuelve | dnsmasq no tiene el registro | Verificar `/etc/dnsmasq.conf` |
| DNS lento | Servidor upstream no responde | Verificar `dnsmasq_dns_servers` |
| `*.k8s.homelab.local` no resuelve | Entrada faltante en dnsmasq | Verificar `address=/.k8s.homelab.local/192.168.1.89` en dnsmasq.conf |
| `*.k8s.homelab.local` resuelve pero no conecta desde WAN | DNAT no configurado | Verificar reglas NAT en `/etc/ufw/before.rules` y ejecutar `playbooks/firewall.yml` |
| Pod no resuelve DNS externo | CoreDNS no puede reenviar | Verificar que dnsmasq acepta queries desde la red de pods (10.42.0.0/16) |
