# Configuración DNS del Homelab

## Arquitectura
```
                    Internet
                        │
                        ▼
              ┌─────────────────┐
              │  Modem/Router   │
              │  192.168.100.1  │
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

macOS usa el DNS del modem (192.168.100.1) por defecto, que no conoce `.homelab.local`.

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

## Troubleshooting

| Problema | Causa | Solución |
|----------|-------|----------|
| `nslookup` funciona pero `ssh nombre` no | macOS no usa el DNS correcto | Crear `/etc/resolver/homelab.local` |
| Nombre no resuelve | dnsmasq no tiene el registro | Verificar `/etc/dnsmasq.conf` |
| DNS lento | Servidor upstream no responde | Verificar `dnsmasq_dns_servers` |
