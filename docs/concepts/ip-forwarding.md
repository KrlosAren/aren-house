# IP Forwarding

## ¿Qué es?

IP Forwarding (reenvío de paquetes IP) es la capacidad de un sistema para actuar como router, reenviando paquetes de red entre diferentes interfaces de red.

Por defecto, Linux **no** reenvía paquetes entre interfaces. Esto significa que si un paquete llega por `eth0` destinado a otra red, Linux lo descarta en lugar de enviarlo por otra interfaz.

## ¿Por qué lo necesitamos?

En el homelab, el gateway (rp1-master) tiene dos interfaces:
- `enx00e04c683da2` (USB-Ethernet) → conectado al modem (WAN)
- `eth0` → conectado al switch interno (LAN)

Sin IP forwarding:
```
Nodo (10.0.0.2) → paquete a 8.8.8.8 → llega a gateway → DESCARTADO
```

Con IP forwarding:
```
Nodo (10.0.0.2) → paquete a 8.8.8.8 → llega a gateway → reenvía por WAN → internet
```

## Cómo funciona

### Verificar estado actual

```bash
cat /proc/sys/net/ipv4/ip_forward
# 0 = deshabilitado
# 1 = habilitado
```

### Habilitar temporalmente

```bash
sudo sysctl -w net.ipv4.ip_forward=1
```

### Habilitar permanentemente

```bash
# Crear archivo de configuración
echo "net.ipv4.ip_forward = 1" | sudo tee /etc/sysctl.d/99-ip-forward.conf

# Aplicar
sudo sysctl -p /etc/sysctl.d/99-ip-forward.conf
```

## Diagrama de flujo

```
┌─────────────────────────────────────────────────────────────┐
│                        GATEWAY                               │
│                                                              │
│   eth0 (10.0.0.1)              enx00e04c683da2 (WAN)        │
│        │                              │                      │
│        │    ┌──────────────────┐      │                      │
│        └───▶│  IP FORWARDING   │──────┘                      │
│             │  (kernel)        │                             │
│             └──────────────────┘                             │
│                     │                                        │
│                     ▼                                        │
│             ┌──────────────────┐                             │
│             │      NAT         │                             │
│             │  (iptables)      │                             │
│             └──────────────────┘                             │
└─────────────────────────────────────────────────────────────┘
        ▲                                      │
        │                                      ▼
   Paquete de                            Paquete a
   nodo interno                          internet
   (10.0.0.2)                           (con IP del gateway)
```

## IP Forwarding vs NAT

Son cosas diferentes pero complementarias:

| Concepto | Función |
|----------|---------|
| **IP Forwarding** | Permite que el kernel reenvíe paquetes entre interfaces |
| **NAT** | Modifica las direcciones IP de los paquetes reenviados |

Necesitas **ambos** para que los nodos internos accedan a internet:
1. IP Forwarding permite que el paquete pase del nodo al internet
2. NAT cambia la IP origen (10.0.0.2 → IP del gateway) para que la respuesta pueda volver

## IPv6

También existe IP forwarding para IPv6:

```bash
# Verificar
cat /proc/sys/net/ipv6/conf/all/forwarding

# Habilitar
echo "net.ipv6.conf.all.forwarding = 1" | sudo tee -a /etc/sysctl.d/99-ip-forward.conf
```

## Seguridad

IP forwarding sin firewall significa que todo el tráfico pasa sin restricciones. Por eso es importante:

1. Habilitar IP forwarding (para que funcione el routing)
2. Configurar firewall (para controlar qué tráfico pasa)

Ver: [Guía de Firewall](../guides/firewall.md)

## En Ansible

En el rol `wireguard`, habilitamos IP forwarding:

```yaml
- name: Enable IP forwarding
  sysctl:
    name: net.ipv4.ip_forward
    value: "1"
    state: present
    sysctl_file: /etc/sysctl.d/99-ip-forward.conf
    reload: yes
```

## Troubleshooting

### Los nodos no tienen internet aunque IP forwarding está habilitado

1. Verificar NAT:
   ```bash
   sudo iptables -t nat -L POSTROUTING -n -v
   ```

2. Verificar que el gateway tiene internet:
   ```bash
   ping 8.8.8.8
   ```

3. Verificar rutas:
   ```bash
   ip route show
   ```

### IP forwarding se desactiva después de reiniciar

Verificar que existe el archivo de configuración:
```bash
cat /etc/sysctl.d/99-ip-forward.conf
```

Si no existe, re-ejecutar el playbook de Ansible.
