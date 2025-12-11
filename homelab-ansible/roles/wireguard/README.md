# Role: wireguard

Instala y configura WireGuard como servidor VPN para el homelab.

## Funciones

| Servicio | Descripción |
|----------|-------------|
| VPN | Túnel seguro para acceso remoto a la red del homelab |
| IP Forwarding | Habilita enrutamiento de paquetes entre interfaces |
| Firewall | Configura reglas iptables para el forwarding de tráfico VPN |

## Variables

| Variable | Default | Descripción |
|----------|---------|-------------|
| `wireguard_address` | `10.0.1.1/24` | Dirección IP y máscara de la interfaz WireGuard |
| `wireguard_port` | `51820` | Puerto UDP de escucha |
| `wireguard_interface` | `wg0` | Nombre de la interfaz WireGuard |
| `wireguard_peers` | `[]` | Lista de peers (clientes) a configurar |

## Ejemplo de uso
```yaml
- hosts: gateway
  become: yes
  vars:
    wireguard_address: "10.0.1.1/24"
    wireguard_port: 51820
    wireguard_peers:
      - public_key: "ABC123PublicKeyDelCliente..."
        allowed_ips: "10.0.1.2/32"
      - public_key: "DEF456OtraPublicKey..."
        allowed_ips: "10.0.1.3/32"
  roles:
    - wireguard
```

## Verificación
```bash
# Estado del servicio
sudo systemctl status wg-quick@wg0

# Ver interfaz y peers conectados
sudo wg show

# Ver clave pública del servidor (para configurar clientes)
cat /etc/wireguard/public.key

# Ver logs
sudo journalctl -u wg-quick@wg0 -f
```

## Dependencias

Las claves (privada y pública) se generan automáticamente si no existen en `/etc/wireguard/`.
