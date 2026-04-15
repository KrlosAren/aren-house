# 014. Migración de netboot a SSD local en todos los nodos

**Fecha:** 2026-04-13
**Estado:** Aceptado
**Supersede:** [ADR-006 Netboot vs Boot Local](006-netboot-vs-local.md)

## Contexto

Los nodos worker (rp2, rp3) operaban con netboot (PXE/NFS) desde rp1-master. Si bien funcionó durante la fase inicial del homelab, en la práctica surgieron varios problemas que llevaron a reconsiderar esta decisión:

- **Fragilidad del boot**: Cualquier interrupción de red durante el boot dejaba el nodo inaccesible
- **Dependencia total de rp1**: Si rp1 caía, los nodos no podían arrancar
- **Complejidad operacional**: Actualizar el OS requería sincronizar `/srv/nfs/{rp2,rp3}/` y archivos TFTP
- **Performance I/O limitado**: Todo el I/O del filesystem pasaba por la red (NFS)
- **overlay2 no funciona sobre NFS**: Docker requería configuración especial con driver `vfs`
- **Debugging complejo**: Problemas de red durante boot eran difíciles de diagnosticar

Además, ambos nodos (rp2, rp3) ya tenían SSDs de 500GB conectados que se usaban solo para Docker y k3s — el OS seguía en NFS.

## Decisión

Migrar cada nodo a **boot local desde SSD**, instalando Ubuntu Server directamente en el SSD de cada nodo.

## Consecuencias

### Positivas
- **Independencia**: Cada nodo puede arrancar sin rp1
- **Performance**: I/O del filesystem local, overlay2 funciona nativamente
- **Simplicidad operacional**: Cada nodo es una máquina Ubuntu estándar
- **Estabilidad**: Sin dependencias de red durante el boot
- **IPs estáticas via netplan**: Ya no se depende de DHCP para arrancar (ver ADR-015)

### Negativas
- **Gestión descentralizada**: Actualizaciones deben aplicarse nodo por nodo (mitigado con Ansible)
- **Reinstalación manual**: Requiere acceso físico o imagen USB para reinstalar
- **TFTP/NFS siguen en rp1**: La infraestructura de netboot permanece pero sin uso activo

## Implementación

- Instalar Ubuntu Server 24.04 LTS en el SSD de cada nodo vía USB booteable
- Configurar IP estática en cada nodo via netplan (ver ADR-015)
- Mantener playbooks de Ansible para configuración inicial: `common.yml`, `firewall.yml`, `k3s.yml`
- Los directorios NFS (`/srv/nfs/{rp2,rp3}/`) se mantienen como referencia pero sin uso

## Historia

| Fecha | Estado |
|-------|--------|
| 2025-12-15 | ADR-006: Adopción de netboot |
| 2026-01 | Identificados problemas de performance con NFS |
| 2026-04 | rp3 pierde conectividad SSH; diagnóstico revela fragilidad del modelo netboot |
| 2026-04-13 | Migración a SSD local completada en rp2 y rp3 |

## Referencias

- [ADR-006](006-netboot-vs-local.md) — decisión original de netboot
- [ADR-007](007-docker-storage-overlay.md) — storage local para Docker
- [ADR-015](015-static-ip-netplan.md) — IPs estáticas via netplan
