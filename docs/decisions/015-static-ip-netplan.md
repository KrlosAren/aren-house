# 015. IPs estáticas via netplan en nodos worker

**Fecha:** 2026-04-13
**Estado:** Aceptado
**Relacionado:** [ADR-003 dnsmasq DHCP/DNS/TFTP](003-dnsmasq-dhcp-dns-tftp.md), [ADR-014 SSD local](014-ssd-local-over-netboot.md)

## Contexto

Con el modelo netboot, los nodos obtenían IP via DHCP de dnsmasq en rp1 como parte del proceso de boot (PXE → DHCP → NFS mount). Era un flujo integrado y necesario.

Al migrar a SSD local (ADR-014), los nodos ya no necesitan DHCP para bootear. Sin embargo, dnsmasq seguía siendo el servidor DHCP para asignar `10.0.0.2` y `10.0.0.3`.

Durante abril 2026, rp3 dejó de obtener IP via DHCP por un problema complejo en rp1:

- dnsmasq generaba DHCPOFFER correctamente (confirmado en logs)
- Los OFFERs no salían físicamente por `eth0` al wire
- Causa raíz: el kernel de rp1 rutea `255.255.255.255` (limited broadcast) por `enx00e04c683da2` (WAN) en vez de `eth0` (LAN), debido a la interacción entre k3s/kube-router, nftables y la tabla de routing `local`
- Se intentaron múltiples fixes: `bind-interfaces`, `dhcp-broadcast`, policy routing con fwmark, nftables DNAT — ninguno fue suficientemente estable

La solución definitiva fue eliminar la dependencia de DHCP en los nodos completamente.

## Decisión

Configurar **IPs estáticas via netplan** en cada nodo worker, eliminando la dependencia de dnsmasq DHCP para la conectividad de los nodos.

## Configuración

**rp2-node** (`/etc/netplan/01-network.yaml`):
```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - 10.0.0.2/24
      routes:
        - to: default
          via: 10.0.0.1
      nameservers:
        addresses:
          - 10.0.0.1
        search:
          - homelab.local
```

**rp3-node** (`/etc/netplan/01-network.yaml`):
```yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - 10.0.0.3/24
      routes:
        - to: default
          via: 10.0.0.1
      nameservers:
        addresses:
          - 10.0.0.1
        search:
          - homelab.local
```

## Consecuencias

### Positivas
- **Sin dependencia de DHCP**: Los nodos arrancan con IP inmediatamente
- **Eliminación del problema de broadcast**: El bug de routing de `255.255.255.255` en rp1 ya no afecta a los nodos
- **Arranque más rápido**: Sin espera de handshake DHCP (DISCOVER→OFFER→REQUEST→ACK)
- **Predecible**: La IP siempre es la misma, sin riesgo de cambio

### Negativas
- **Gestión manual de IPs**: Cambios de IP requieren editar netplan en cada nodo
- **dnsmasq DHCP sigue corriendo**: Aún gestiona el switch y otros dispositivos futuros en la LAN

## Notas

- dnsmasq mantiene entradas `dhcp-host` para rp2 y rp3 por compatibilidad, pero los nodos ya no hacen DHCP requests
- rp1-master mantiene IP estática `10.0.0.1` en `eth0` via netplan (nunca dependió de DHCP en LAN)
- La IP WAN de rp1 (`enx00e04c683da2`) sigue siendo dinámica via DHCP del ISP

## Referencias

- [ADR-003](003-dnsmasq-dhcp-dns-tftp.md) — dnsmasq como servidor DHCP/DNS
- [ADR-014](014-ssd-local-over-netboot.md) — migración a SSD local
