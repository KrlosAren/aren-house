# Conceptos

Explicaciones de los conceptos técnicos utilizados en el homelab. Útil como referencia de aprendizaje.

## Índice

### Networking
| Concepto | Descripción |
|----------|-------------|
| [DHCP](dhcp.md) | Asignación automática de IPs |
| [DNS](dns.md) | Resolución de nombres de dominio |
| [TFTP](tftp.md) | Transferencia de archivos para boot |
| [PXE](pxe.md) | Boot por red sin almacenamiento local |
| [NAT](nat.md) | Traducción de direcciones de red |
| [IP Forwarding](ip-forwarding.md) | Routing entre interfaces de red |
| [iptables](iptables-basics.md) | Firewall y reglas de red en Linux |
| [Netplan](netplan.md) | Configuración de red en Ubuntu |

### Seguridad
| Concepto | Descripción |
|----------|-------------|
| [VPN](vpn.md) | Túneles seguros de red |
| [WireGuard](wireguard.md) | Protocolo VPN moderno |
| [UFW](ufw.md) | Uncomplicated Firewall para Linux |

### Sistemas
| Concepto | Descripción |
|----------|-------------|
| [systemd](systemd.md) | Sistema de init y gestión de servicios |
| [NFS](nfs.md) | Network File System para compartir archivos |

### Raspberry Pi
| Concepto | Descripción |
|----------|-------------|
| [EEPROM](raspberry-pi-eeprom.md) | Configuración de boot en Raspberry Pi |

### Kubernetes
Para conceptos de Kubernetes/k3s, ver:
- [K3s Setup](../k3s-setup.md) - Guía de instalación y configuración
- [K3s Guide](../guides/k3s-guide.md) - Guía operacional
- [ADR-012: K3s sobre k8s vanilla](../decisions/012-k3s-over-k8s.md) - Por qué k3s

## Cómo agregar conceptos

1. Crear archivo `concepto.md` en esta carpeta
2. Seguir la estructura:
   - **Qué es**: Definición simple
   - **Para qué sirve**: Uso práctico
   - **Cómo funciona**: Explicación técnica (opcional)
   - **En el homelab**: Cómo lo usamos aquí
   - **Comandos útiles**: Referencia rápida
3. Agregar entrada en este índice
