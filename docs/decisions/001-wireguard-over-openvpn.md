# 001. WireGuard sobre OpenVPN

**Fecha:** 2025-12-06
**Estado:** Superseded por [ADR-008: Tailscale para CGNAT](008-tailscale-cgnat.md)

## Contexto

Necesito acceso remoto a la red interna del homelab (10.0.0.0/24) desde mi Mac sin tener que hacer doble salto SSH (primero al gateway, luego a la máquina interna).

Las opciones principales son:
- **OpenVPN**: Solución tradicional, muy probada
- **WireGuard**: Protocolo moderno, integrado en el kernel de Linux

## Decisión

Usar **WireGuard** como solución VPN.

## Consecuencias

### Positivas
- **Rendimiento**: Corre en el kernel de Linux, menor overhead que OpenVPN (userspace)
- **Simplicidad**: Configuración mínima (~15 líneas vs cientos en OpenVPN)
- **Criptografía moderna**: ChaCha20, Curve25519, sin opciones inseguras que elegir
- **Menor superficie de ataque**: ~4,000 líneas de código vs ~100,000 de OpenVPN
- **Roaming**: Cambia de red (WiFi a móvil) sin reconectar

### Negativas
- **Menos flexible**: No hay opciones de configuración avanzadas
- **IPs fijas**: Cada peer necesita IP asignada manualmente (no hay DHCP)
- **Menos documentación legacy**: OpenVPN tiene más recursos/tutoriales antiguos

### Notas
- Puerto UDP 51820 por defecto
- Las llaves se generan con `wg genkey` y `wg pubkey`
- La llave pública de cada lado va en el `[Peer]` del otro

## Actualización (2026-01)

WireGuard fue reemplazado por **Tailscale** como solución primaria de VPN debido a que el ISP usa CGNAT, lo que impide conexiones entrantes al puerto 51820. WireGuard queda como solución de backup/legacy para cuando se tenga IP pública.

Ver [ADR-008: Tailscale para CGNAT](008-tailscale-cgnat.md) y [ADR-009: Workaround CGNAT](009-cgnat-workaround.md).
