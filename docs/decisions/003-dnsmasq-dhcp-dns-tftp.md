# 003. dnsmasq como servidor DHCP, DNS y TFTP

**Fecha:** 2025-12-08
**Estado:** Aceptado

## Contexto

El homelab necesita:
- Asignar IPs fijas a los dispositivos de la red interna (10.0.0.0/24)
- Resolver nombres locales (rp2.homelab.local → 10.0.0.2)
- Servir archivos de boot para PXE (netboot de Raspberry Pi sin microSD)

Opciones consideradas:
- **Servicios separados**: isc-dhcp-server + bind9 + tftpd-hpa
- **dnsmasq**: Solución integrada DHCP + DNS + TFTP

## Decisión

Usar **dnsmasq** como servidor unificado para DHCP, DNS y TFTP.

## Consecuencias

### Positivas
- **Simplicidad**: Un solo servicio, un solo archivo de configuración
- **Integración DHCP-DNS**: Los hostnames asignados por DHCP se resuelven automáticamente
- **Bajo consumo**: Ideal para Raspberry Pi
- **PXE boot**: TFTP integrado simplifica el netboot

### Negativas
- **Menos flexible**: Para configuraciones DNS complejas, bind9 es más potente
- **Sin interfaz gráfica**: Pi-hole podría agregarse después para dashboard y bloqueo de ads

## Configuración clave
```
interface=eth0           # Solo red interna
dhcp-range=10.0.0.100,10.0.0.200,24h
dhcp-host=MAC,nombre,IP  # IPs fijas
enable-tftp
tftp-root=/srv/tftp
domain=homelab.local
```

## Servicios proporcionados

| Servicio | Puerto | Función |
|----------|--------|---------|
| DHCP | 67/udp | Asignación de IPs |
| DNS | 53/udp | Resolución de nombres |
| TFTP | 69/udp | Archivos de boot PXE |
