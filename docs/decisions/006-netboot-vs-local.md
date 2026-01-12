# 006. Netboot vs Boot Local

**Fecha:** 2025-12-15
**Estado:** Aceptado

## Contexto

Necesitamos decidir cómo bootean los nodos worker (rp2, rp3) del homelab.

Opciones:
1. **Boot local**: Cada nodo tiene su propio OS en microSD/SSD
2. **Netboot (PXE/NFS)**: Los nodos bootean por red desde el gateway

## Decisión

Usar **netboot** para los nodos worker.

## Consecuencias

### Positivas
- **Gestión centralizada**: Un solo lugar para actualizar el OS
- **Fácil reinstalación**: Solo recrear el directorio NFS
- **Consistencia**: Todos los nodos tienen la misma configuración base
- **Aprendizaje**: Entender DHCP, TFTP, NFS a fondo

### Negativas
- **Dependencia del gateway**: Si rp1 cae, los nodos no pueden bootear
- **Complejidad inicial**: Más configuración que boot local
- **Performance**: Boot más lento que local
- **Limitaciones NFS**: overlay2 de Docker no funciona sobre NFS

## Implementación

- Configurar dnsmasq como DHCP/TFTP server
- Configurar NFS server para filesystems en `/srv/nfs/{rp2,rp3}/`
- TFTP sirve kernel e initrd desde `/srv/tftp/{serial}/`
- Necesitar storage local para Docker (ver [ADR-007](007-docker-storage-overlay.md))

## Referencias

- [netboot-concepts.md](../netboot-concepts.md)
- [netboot-node-setup.md](../netboot-node-setup.md)
