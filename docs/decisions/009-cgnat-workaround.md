# 009. Workaround para CGNAT del ISP

**Fecha:** 2025-12-15
**Estado:** Aceptado

## Contexto

El ISP (Entel Chile) usa CGNAT (Carrier-Grade NAT), lo que significa:

1. La IP pública es compartida entre múltiples clientes
2. El modem recibe una IP privada del rango 100.64.0.0/10 (CGNAT)
3. Port forwarding configurado en el modem no tiene efecto

### Evidencia

```bash
# IP del modem (WAN) - visible en interfaz del modem
100.78.117.90  ← Rango CGNAT (100.64.0.0/10)

# IP pública real (compartida con otros)
curl ifconfig.me → 200.111.224.99
```

### Cómo detectar CGNAT

```bash
# Si la IP WAN del modem está en estos rangos = CGNAT
100.64.0.0 - 100.127.255.255  (100.64.0.0/10)
```

## Impacto

- ❌ WireGuard directo no funciona (puerto 51820 no llega)
- ❌ Port forwarding en modem es inútil
- ❌ No se pueden exponer servicios HTTP/HTTPS directamente
- ❌ DuckDNS apunta a IP compartida (no sirve para conexiones entrantes)

## Soluciones Implementadas

### 1. Tailscale para VPN (principal)
- NAT traversal automático
- Funciona sin abrir puertos
- Ver [ADR-008](008-tailscale-cgnat.md)

### 2. DuckDNS (mantenido)
- Actualiza IP pública cada 5 minutos
- Útil si conseguimos IP pública en el futuro
- Útil para Cloudflare Tunnel (conexión saliente)

## Soluciones Futuras (no implementadas)

### Cloudflare Tunnel
- Para exponer servicios HTTP/HTTPS públicamente
- Conexión saliente desde homelab → Cloudflare
- No requiere IP pública
- **Estado**: Pendiente implementar

### Pedir IP pública a Entel
- Solución definitiva para WireGuard directo
- **Estado**: Pendiente evaluar costo

### VPS como relay
- Servidor externo ($5/mes) como punto de entrada
- **Estado**: No necesario con Tailscale gratuito

## Lecciones Aprendidas

1. **Siempre verificar CGNAT antes de configurar VPN**
   - Revisar IP WAN del modem antes de asumir que port forwarding funcionará

2. **Conexiones salientes siempre funcionan**
   - Tailscale, Cloudflare Tunnel, SSH reverso aprovechan esto

3. **DuckDNS sigue siendo útil**
   - Aunque no sirva para WireGuard, sirve para identificar la IP pública

4. **ISPs de fibra residencial frecuentemente usan CGNAT**
   - Es común en Chile y Latinoamérica

## Referencias

- [ADR-008: Tailscale vs WireGuard](008-tailscale-cgnat.md)
- [tailscale-setup.md](../guides/tailscale-setup.md)
