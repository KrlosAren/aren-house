# ADR-017: `tailscale serve` en vez de Ingress para servicios sensibles

**Estado:** Aceptado (actualizado 2026-11-22: Tailscale Services)
**Fecha:** 2026-11-21

## Contexto

El patrón estándar del cluster para exponer apps es `Ingress` (Traefik k3s) con host `*.k8s.homelab.local`:

```
Cliente → DNS → 192.168.1.89 (WAN rp1) → DNAT :80/:443 → MetalLB 10.0.0.50 → Traefik → Pod
```

Al desplegar **go2rtc** (stream de la cámara Tapo C110) y **ntfy** (notificaciones push) aparecieron tres problemas con este patrón:

1. **Exposición a toda la red de la casa.** La IP WAN de rp1 (192.168.1.89) vive en la red del módem Movistar (WiFi "casona"). Cualquier dispositivo conectado ahí — incluidos invitados — puede alcanzar cualquier Ingress con solo enviar el header `Host` correcto. Se verificó: `curl -H "Host: camara.k8s.homelab.local" http://192.168.1.89/api/streams` devolvía `200` con la lista de streams.
2. **No funciona desde fuera de casa.** La regla DNAT en `/etc/ufw/before.rules` solo matchea `-i enx00e04c683da2` (WAN). El tráfico que llega por `tailscale0` no se redirige a MetalLB.
3. **Basic auth no es usable en móvil.** Se agregó basic auth a go2rtc como mitigación, pero Safari/iOS no guarda credenciales de basic auth en sitios HTTP: había que escribir la contraseña en cada acceso. Además, sin TLS (Cert-Manager pendiente) la contraseña viaja en texto plano por el WiFi.

Alternativas evaluadas:

| Opción | Descartada porque |
|---|---|
| Basic auth en la app / middleware Traefik | No persiste en móvil, texto plano sin TLS, sigue sin funcionar fuera de casa |
| `ipAllowList` en Traefik | `externalTrafficPolicy: Cluster` hace SNAT al saltar entre nodos → Traefik no ve la IP real del cliente. Cambiarlo a `Local` afecta a todo el cluster |
| Forward-auth (tinyauth/Authelia) | Componente extra que mantener; sigue sin funcionar fuera de casa y sin TLS |
| Tailscale Kubernetes Operator | Más piezas (operator, OAuth client, CRDs) para dos servicios; evaluar si crece la cantidad |
| Tailscale Funnel | Expone a internet público — lo contrario de lo buscado |

## Decisión

Los servicios **sensibles o sin autenticación propia** se exponen con `tailscale serve` en rp1, **sin Ingress**:

```
Dispositivo del tailnet → https://rp1-master.tailb0ca85.ts.net:<puerto>
                        → tailscaled en rp1 (termina TLS)
                        → http://<clusterIP fija>:<puerto>  (kube-proxy en rp1)
                        → Pod
```

Reglas:
- El `Service` declara `spec.clusterIP` fija: rp1 alcanza ClusterIPs vía kube-proxy pero no resuelve el DNS del cluster, y una IP reasignada rompería el proxy silenciosamente.
- Puertos `8443+` en rp1 (el `:443` lo ocupa Traefik Docker en `0.0.0.0`).
- La configuración se gestiona en `homelab-ansible/playbooks/tailscale.yml` (variable `tailscale_serve`, tag `serve`), idempotente: solo agrega puertos que no estén ya publicados.
- Requiere "HTTPS Certificates" habilitado en el admin de Tailscale (DNS).

Asignación actual:

| Puerto | App | Destino |
|---|---|---|
| 8443 | go2rtc (ns `cameras`) | `10.43.92.177:1984` |
| 8444 | ntfy (ns `ntfy`) | `10.43.200.10:80` |

El Ingress sigue siendo válido para apps no sensibles o con login propio robusto.

## Consecuencias

**Positivo:**
- Solo dispositivos autenticados en el tailnet acceden ("identidad del dispositivo" en vez de "estar en la red correcta").
- Funciona igual dentro y fuera de casa, incluso detrás del CGNAT (túnel WireGuard directo o vía DERP).
- TLS válido (Let's Encrypt vía `*.ts.net`) sin depender de Cert-Manager.
- go2rtc puede correr sin auth propia → acceso sin fricción desde el teléfono.
- Invitados en "casona" ya no ven la cámara (verificado: `404` vía 192.168.1.89).

**Negativo:**
- **rp1 es punto único de falla** para estos servicios (ya lo es como gateway).
- El teléfono necesita la app Tailscale activa.
- Los nombres de máquina del tailnet quedan en logs públicos de Certificate Transparency (solo nombres, no acceso).
- Configuración fuera de k8s: `kubectl apply` no basta, hay que correr el playbook. Mitigado documentándolo en `CLAUDE.md`.
- ClusterIPs fijas deben elegirse a mano evitando colisiones dentro de `10.43.0.0/16`.

**Revisar si:**
- Se superan ~5 servicios publicados así → evaluar el Tailscale Kubernetes Operator.
- Se despliega Cert-Manager → reevaluar Ingress + forward-auth para apps de uso doméstico (no solo del dueño).

---

## Actualización 2026-11-22: Tailscale Services (un nombre por servicio)

Con 6 servicios publicados (el umbral de "revisar si" era ~5), se reemplazaron los puertos `rp1-master.<tailnet>.ts.net:8443-8448` por **Tailscale Services**: cada servicio recibe su propia IP virtual del tailnet (TailVIP) y su nombre, todos en `:443`.

| Antes | Ahora |
|---|---|
| `https://rp1-master.tailb0ca85.ts.net:8445` | `https://frigate.tailb0ca85.ts.net` |

Se eligió sobre el **Tailscale Kubernetes Operator** porque no agrega pods (un proxy por servicio) ni OAuth client: rp1 sigue siendo el único host, y encaja con el playbook existente.

**Cambios necesarios:**
- rp1 pasó a identidad de máquina **`tag:homelab`** (requisito para ser host de Services). Efecto colateral positivo: la llave de rp1 ya no vence (antes: 180 días, y su vencimiento habría tumbado todas las URLs).
- Política de acceso: grant `autogroup:member → svc:*` (ip 443) y `autoApprovers` para los servicios y la ruta `10.0.0.0/24`.
- Playbook: lista `tailscale_services` (reemplaza a `tailscale_serve`). Los puertos 8443-8448 fueron retirados de rp1 (verificado 2026-11-25: `tailscale serve status` solo muestra los 6 Services).
- Clientes con la URL vieja guardada dejan de funcionar **en silencio**. En la app ntfy de iOS hubo que re-suscribir los topics: el hash del topic upstream en ntfy.sh depende de `base-url`.

**Lección:** consola (definición), política (autorización) y host (anuncio) no se validan entre sí. Un nombre distinto (`camera` vs `camara`) produce, sin error, un servicio sin DNS o con DNS pero sin conexión.

**Sigue vigente:** rp1 como punto único de falla y la configuración fuera de k8s.
