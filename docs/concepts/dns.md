# DNS (Domain Name System)

## Qué es

Sistema que traduce nombres de dominio legibles (como `google.com`) a direcciones IP (como `142.250.80.14`). Es como la "agenda de contactos" de internet.

## Para qué sirve

- **Humanos**: Recordamos nombres, no números
- **Máquinas**: Solo entienden IPs
- **DNS**: Traduce entre ambos mundos

```
Usuario escribe: github.com
       ↓
DNS resuelve: 140.82.121.3
       ↓
Navegador conecta a: 140.82.121.3
```

## Cómo funciona

### Jerarquía DNS

```
                    [Root DNS (.)]
                         |
         +---------------+---------------+
         |               |               |
      [.com]          [.org]          [.local]
         |               |               |
    [google.com]   [wikipedia.org]  [homelab.local]
         |                               |
   [www.google.com]              [rp2.homelab.local]
```

### Tipos de registros comunes

| Tipo | Propósito | Ejemplo |
|------|-----------|---------|
| A | Nombre → IPv4 | `rp2.homelab.local → 10.0.0.2` |
| AAAA | Nombre → IPv6 | `rp2 → fe80::1` |
| CNAME | Alias | `www → rp2.homelab.local` |
| PTR | IP → Nombre (reverso) | `10.0.0.2 → rp2.homelab.local` |
| MX | Servidor de correo | `mail.homelab.local` |

### Proceso de resolución

1. **Cache local**: ¿Ya lo busqué antes?
2. **Servidor DNS configurado**: Pregunta a dnsmasq/router
3. **DNS recursivo**: Si no sabe, pregunta a servidores externos
4. **Respuesta**: Se guarda en cache para futuras consultas

## Puertos

| Puerto | Protocolo | Uso |
|--------|-----------|-----|
| 53/udp | DNS queries | Consultas normales (< 512 bytes) |
| 53/tcp | DNS queries | Respuestas grandes, transferencias de zona |

## En el homelab

**dnsmasq** actúa como DNS local:

```bash
# /etc/dnsmasq.conf
domain=homelab.local
local=/homelab.local/          # No reenviar .homelab.local a internet
expand-hosts                    # Agregar dominio automáticamente

# Los hosts de /etc/hosts se resuelven automáticamente
# /etc/hosts:
# 10.0.0.1  gateway
# 10.0.0.2  rp2
```

Además, dnsmasq reenvía consultas externas (google.com) a DNS públicos.

Ver: [ADR-003 dnsmasq](../decisions/003-dnsmasq-dhcp-dns-tftp.md)

## Comandos útiles

```bash
# Resolver un nombre
dig rp2.homelab.local
nslookup rp2.homelab.local

# Ver servidor DNS configurado
cat /etc/resolv.conf

# Limpiar cache DNS (systemd)
sudo systemd-resolve --flush-caches

# Consultar un servidor DNS específico
dig @10.0.0.1 rp2.homelab.local

# Ver registros de dnsmasq
sudo journalctl -u dnsmasq -f
```
