# Architecture Decision Records (ADRs)

Documentación de decisiones técnicas importantes del homelab.

## ¿Qué es un ADR?

Un ADR es un documento corto que captura una decisión arquitectónica importante junto con su contexto y consecuencias. Útil para recordar el "por qué" detrás de cada decisión.

## Formato

Cada ADR sigue esta estructura:

```markdown
# [Número]. [Título]

**Fecha:** YYYY-MM-DD
**Estado:** Propuesto | Aceptado | Deprecado | Reemplazado

## Contexto
¿Qué problema estamos tratando de resolver?

## Decisión
¿Qué decidimos hacer?

## Consecuencias
¿Qué implica esta decisión? (positivo y negativo)
```

## Índice

| # | Decisión | Estado | Fecha |
|---|----------|--------|-------|
| 001 | [WireGuard sobre OpenVPN](001-wireguard-over-openvpn.md) | Aceptado | 2025-12-06 |
| 002 | [Segmentación de red con Raspberry Pi](002-network-segmentation.md) | Aceptado | 2025-12-06 |
| 003 | [dnsmasq como DHCP, DNS y TFTP](003-dnsmasq-dhcp-dns-tftp.md) | Aceptado | 2025-12-06 |
| 004 | [IP Forwarding y NAT](004-ip-forwarding-nat.md) | Aceptado | 2025-12-06 |
| 005 | [UFW como Firewall](005-ufw-firewall.md) | Aceptado | 2025-12-06 |
| 006 | [Netboot vs Boot Local](006-netboot-vs-local.md) | Aceptado | 2025-12-15 |
| 007 | [Docker Storage: overlay2 en disco local](007-docker-storage-overlay.md) | Aceptado | 2025-12-15 |
| 008 | [Tailscale vs WireGuard (CGNAT)](008-tailscale-cgnat.md) | Aceptado | 2025-12-15 |
| 009 | [Workaround para CGNAT](009-cgnat-workaround.md) | Aceptado | 2025-12-15 |
