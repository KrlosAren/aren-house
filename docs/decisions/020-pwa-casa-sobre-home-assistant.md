# ADR-020: PWA propia "casa" (Go + React) en vez de Home Assistant

**Estado:** Aceptado
**Fecha:** 2026-11-26

## Contexto

Con la cámara (Frigate), los Atajos de iOS para armar/desarmar, dos focos WiZ con control local y la alarma visual, faltaba **un lugar para ver y controlar todo** desde el teléfono. Opciones evaluadas:

| | Home Assistant | PWA propia (`apps/casa`) |
|---|---|---|
| Integraciones | +2000 (WiZ, Frigate, Tapo, MQTT…) | Solo lo que se programe |
| Presencia multi-persona | Nativa (app companion) | No (Atajos por persona) |
| Despliegue en k3s | Incómodo: `hostNetwork` para descubrir (mDNS, broadcast UDP de WiZ); la versión Container no tiene add-ons | Un pod stateless, multi-arch, sin privilegios |
| Mantenimiento | Releases mensuales, base de datos, backups | Un binario Go + estáticos |
| Aprendizaje | Configurar una plataforma | Go, React/PWA, CI multi-arch, k8s |

Con 1 cámara y 2 focos, HA es más de lo necesario; la lógica difícil (estado real de la cámara, control WiZ, restauración del foco) ya estaba resuelta y probada.

## Decisión

**`apps/casa`**: API en Go (solo librería estándar) que sirve una PWA en React embebida (`go:embed`).

- **API como centro**, PWA como un cliente más: los Atajos de iOS pueden migrar a `/api/camara/deteccion` y dejar de depender de detalles de Frigate (`ON`/`OFF` en mayúsculas, endpoint).
- Fuentes: Frigate `/api/stats` (estado REAL de detección, no MQTT), WiZ por UDP :38899, Prometheus (nodos, temperatura, ping de la cámara). Cada fuente falla por separado.
- **Sin auth propia**: acceso solo por Tailscale Service `svc:casa` (ADR-017), que además da el HTTPS que una PWA exige para instalarse.
- Imagen multi-arch en **GHCR** construida por `.github/workflows/casa.yml` en runners de GitHub (el registry local no está desplegado). Despliegue manual por tag = SHA.
- `frigate-estado` y `alarma-visual` siguen como pods separados: si la API falla, las alertas siguen funcionando.

## Consecuencias

- Cada dispositivo nuevo requiere código (no hay "agregar integración").
- **Revisar si**: se agregan muchos dispositivos de marcas distintas, se necesita presencia de varias personas, o reglas que combinen muchas condiciones → reconsiderar Home Assistant (podría convivir: usa el mismo Mosquitto).
- Si la imagen GHCR queda privada, hace falta un `imagePullSecret` en el namespace `casa`.
