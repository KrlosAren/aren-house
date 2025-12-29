# Iptables Básico

## ¿Qué es?

Iptables es el firewall del kernel de Linux. Permite filtrar, modificar y redirigir paquetes de red según reglas definidas.

## Conceptos Fundamentales

### Tablas

Iptables organiza las reglas en **tablas**, cada una con un propósito diferente:

| Tabla | Propósito | Uso común |
|-------|-----------|-----------|
| `filter` | Filtrar paquetes (permitir/denegar) | Firewall |
| `nat` | Traducción de direcciones | NAT, port forwarding |
| `mangle` | Modificar paquetes | QoS, TTL |
| `raw` | Excepciones de connection tracking | Optimización |

### Cadenas (Chains)

Dentro de cada tabla hay **cadenas** que determinan cuándo se aplican las reglas:

#### Tabla filter
| Cadena | Cuándo se aplica |
|--------|------------------|
| `INPUT` | Paquetes destinados al host local |
| `OUTPUT` | Paquetes originados en el host local |
| `FORWARD` | Paquetes que pasan a través del host (routing) |

#### Tabla nat
| Cadena | Cuándo se aplica |
|--------|------------------|
| `PREROUTING` | Antes de decidir routing (DNAT) |
| `POSTROUTING` | Después de decidir routing (SNAT/MASQUERADE) |
| `OUTPUT` | Paquetes locales antes de routing |

### Targets (Acciones)

Cada regla termina con una acción:

| Target | Acción |
|--------|--------|
| `ACCEPT` | Permitir el paquete |
| `DROP` | Descartar silenciosamente |
| `REJECT` | Rechazar y notificar al origen |
| `MASQUERADE` | NAT dinámico (para conexiones con IP variable) |
| `SNAT` | NAT estático (IP fija) |
| `DNAT` | Cambiar destino (port forwarding) |
| `LOG` | Registrar en logs |

## Flujo de Paquetes

```
                              ┌─────────────────────┐
                              │    PREROUTING       │
                              │    (nat, mangle)    │
                              └──────────┬──────────┘
                                         │
                                         ▼
                              ┌─────────────────────┐
                              │  Decisión Routing   │
                              └──────────┬──────────┘
                                         │
                    ┌────────────────────┼────────────────────┐
                    │                    │                    │
                    ▼                    ▼                    ▼
         ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
         │      INPUT       │  │     FORWARD      │  │      OUTPUT      │
         │    (filter)      │  │    (filter)      │  │    (filter)      │
         └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
                  │                     │                     │
                  ▼                     │                     │
         ┌──────────────────┐           │                     │
         │  Proceso Local   │           │                     │
         └──────────────────┘           │                     │
                                        │                     │
                                        ▼                     ▼
                              ┌─────────────────────────────────┐
                              │         POSTROUTING             │
                              │         (nat, mangle)           │
                              └─────────────────────────────────┘
                                         │
                                         ▼
                                      Salida
```

## Comandos Básicos

### Ver reglas

```bash
# Ver reglas de filter (por defecto)
sudo iptables -L -n -v

# Ver reglas de NAT
sudo iptables -t nat -L -n -v

# Ver con números de línea
sudo iptables -L -n --line-numbers
```

### Agregar reglas

```bash
# Sintaxis básica
iptables -t TABLA -A CADENA [condiciones] -j TARGET

# Ejemplos:

# Permitir SSH
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Permitir ping
sudo iptables -A INPUT -p icmp -j ACCEPT

# Bloquear IP específica
sudo iptables -A INPUT -s 192.168.1.100 -j DROP

# NAT para red interna
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

### Eliminar reglas

```bash
# Por número de línea
sudo iptables -D INPUT 3

# Por especificación exacta
sudo iptables -D INPUT -p tcp --dport 22 -j ACCEPT
```

### Políticas por defecto

```bash
# Denegar todo por defecto
sudo iptables -P INPUT DROP
sudo iptables -P FORWARD DROP
sudo iptables -P OUTPUT ACCEPT
```

## Reglas en el Homelab

### NAT para nodos internos

```bash
# Los nodos (10.0.0.x) pueden acceder a internet
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

### FORWARD para WireGuard

En el template de WireGuard (`wg0.conf.j2`):
```ini
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -A FORWARD -o wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -D FORWARD -o wg0 -j ACCEPT
```

## Persistencia

Las reglas de iptables se pierden al reiniciar. Para persistirlas:

### Opción 1: iptables-persistent

```bash
sudo apt install iptables-persistent

# Guardar reglas actuales
sudo netfilter-persistent save

# Las reglas se guardan en:
# /etc/iptables/rules.v4
# /etc/iptables/rules.v6
```

### Opción 2: Script en rc.local

```bash
# /etc/rc.local
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enx00e04c683da2 -j MASQUERADE
```

### Opción 3: WireGuard PostUp (lo que usamos)

Las reglas de FORWARD se aplican automáticamente cuando WireGuard inicia.

## Debugging

### Ver paquetes rechazados

```bash
# Agregar logging antes de DROP
sudo iptables -A INPUT -j LOG --log-prefix "IPT-DROP: "

# Ver logs
sudo tail -f /var/log/syslog | grep IPT
```

### Contadores

```bash
# Ver cuántos paquetes coinciden con cada regla
sudo iptables -L -n -v

# Resetear contadores
sudo iptables -Z
```

## nftables (sucesor)

nftables es el sucesor moderno de iptables. En sistemas nuevos puede usarse en lugar de iptables, pero iptables sigue siendo ampliamente usado y documentado.

```bash
# Ver si nftables está activo
sudo nft list ruleset
```

## Referencias

- [Netfilter Documentation](https://www.netfilter.org/documentation/)
- [iptables Tutorial](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html)
