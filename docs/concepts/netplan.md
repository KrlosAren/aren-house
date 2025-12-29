# Netplan

## ¿Qué es?

Netplan es el sistema de configuración de red de Ubuntu. Usa archivos YAML para definir la configuración y luego aplica esa configuración a NetworkManager o systemd-networkd.

## Arquitectura

```
┌────────────────────────────────────────┐
│          Archivos YAML                  │
│    /etc/netplan/*.yaml                  │
└────────────────────┬───────────────────┘
                     │
                     ▼
           ┌─────────────────┐
           │     netplan     │
           │    generate     │
           └────────┬────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌──────────────────┐   ┌──────────────────┐
│ NetworkManager   │   │ systemd-networkd │
│   (desktop)      │   │    (server)      │
└──────────────────┘   └──────────────────┘
```

En Ubuntu Server, netplan usa `systemd-networkd` como backend.

## Ubicación de archivos

```
/etc/netplan/
├── 01-network.yaml       # Configuración principal
├── 50-cloud-init.yaml    # Generado por cloud-init (si existe)
└── 60-eth0.yaml          # Configuración adicional
```

Los archivos se aplican en orden alfabético. Configuraciones posteriores pueden sobrescribir anteriores.

## Sintaxis Básica

```yaml
network:
  version: 2
  renderer: networkd  # o NetworkManager
  ethernets:
    eth0:
      dhcp4: true
```

## Ejemplos de Configuración

### DHCP simple

```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: true
```

### IP estática

```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - 10.0.0.1/24
      routes:
        - to: default
          via: 10.0.0.254
      nameservers:
        addresses:
          - 1.1.1.1
          - 8.8.8.8
```

### Múltiples interfaces (Gateway del homelab)

```yaml
network:
  version: 2
  ethernets:
    # WAN - hacia el modem
    enx00e04c683da2:
      dhcp4: true

    # LAN - hacia el switch interno
    eth0:
      dhcp4: false
      addresses:
        - 10.0.0.1/24
```

### Con DNS personalizado

```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: true
      dhcp4-overrides:
        use-dns: false
      nameservers:
        addresses:
          - 10.0.0.1
        search:
          - homelab.local
```

## Comandos

### Aplicar configuración

```bash
sudo netplan apply
```

### Probar configuración (con timeout)

```bash
sudo netplan try
# Revierte automáticamente si no confirmas en 120 segundos
```

### Generar configuración sin aplicar

```bash
sudo netplan generate
```

### Ver configuración actual

```bash
# Configuración de netplan
cat /etc/netplan/*.yaml

# Estado real de las interfaces
ip addr show
ip route show
```

### Debug

```bash
sudo netplan --debug apply
```

## Integración con dnsmasq

Para que los nodos usen dnsmasq como DNS, configuramos netplan para no usar el DNS del DHCP:

```yaml
# /etc/netplan/60-dns.yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4-overrides:
        use-dns: false
      nameservers:
        addresses:
          - 10.0.0.1
        search:
          - homelab.local
```

## Verificar resolución DNS

```bash
# Ver configuración de resolución
resolvectl status

# Ver configuración por interfaz
resolvectl status eth0
```

## Problemas Comunes

### Error de sintaxis YAML

```bash
# Validar antes de aplicar
sudo netplan generate

# Si hay error, revisar:
# - Indentación (espacios, no tabs)
# - Dos puntos seguidos de espacio
# - Guiones para listas
```

### Interfaz no aparece

```bash
# Verificar que existe
ip link show

# Verificar nombre exacto
ls /sys/class/net/
```

### Cambios no se aplican

```bash
# Forzar recarga de networkd
sudo systemctl restart systemd-networkd

# O reiniciar
sudo reboot
```

### Cloud-init sobrescribe configuración

Si existe `/etc/netplan/50-cloud-init.yaml`, cloud-init puede regenerar la configuración.

```bash
# Deshabilitar network de cloud-init
echo "network: {config: disabled}" | sudo tee /etc/cloud/cloud.cfg.d/99-disable-network-config.cfg
```

## En el Homelab

### Gateway (rp1-master)

Tiene dos interfaces configuradas:
- `enx00e04c683da2`: WAN con DHCP del modem
- `eth0`: LAN con IP fija 10.0.0.1

### Nodos (rp2, rp3)

Configurados con DHCP en eth0:
```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: true
```

El playbook `prepare-node.yml` crea esta configuración si no existe.

## Referencias

- [Netplan Reference](https://netplan.io/reference/)
- [Ubuntu Netplan Documentation](https://ubuntu.com/server/docs/network-configuration)
