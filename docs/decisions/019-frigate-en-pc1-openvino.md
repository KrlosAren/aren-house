# ADR-019: Frigate en pc1-node con detector OpenVINO (CPU)

**Estado:** Aceptado
**Fecha:** 2026-11-23

## Contexto

Frigate corría en rp2-node (Pi 5) con el detector `cpu` (TFLite, 3 threads) y su PV local-path en el SSD de rp2. Con la llegada de pc1-node (ADR-018, i7-4770 + HD 4600) se evaluó moverlo. Mediciones del 2026-11-23 con `frigate:0.18.2` y el stream real de la Tapo C110:

| | rp2 (Pi 5) | pc1-node (i7-4770) |
|---|---|---|
| Inferencia en Frigate (`/api/stats`) | ~63 ms (TFLite) | **~17-26 ms (OpenVINO CPU)** |
| Inferencia pura del modelo (benchmark aislado) | — | TFLite 18,5 ms · OpenVINO 8 ms |
| Decodificar `casona_sub` 720p (CPU / VAAPI) | — | 9 % / 6 % de un core |
| Decodificar `casona` 2304x1296 (CPU / VAAPI) | — | 29 % / 7 % de un core |
| Dispositivos OpenVINO | — | solo `CPU` (GPU no soporta Haswell) |
| Driver VAAPI | — | **i965** (iHD falla en Haswell) |

La inferencia que reporta Frigate incluye IPC y preprocesado; por eso es mayor que el benchmark puro.

## Decisión

- Frigate corre en **pc1-node** (`nodeSelector`), detector **`openvino` con `device: CPU`** y el modelo `ssdlite_mobilenet_v2` incluido en la imagen amd64.
- **Sin VAAPI por ahora.** Solo se decodifica el stream de 720p, y por CPU cuesta ~9 % de un core. Usar la iGPU exige `privileged: true` o el Intel GPU device plugin (el cgroup de dispositivos de containerd bloquea `/dev/dri`). Se activará si se detecta sobre el stream principal o se agregan cámaras (`LIBVA_DRIVER_NAME=i965`, `preset-vaapi`).
- Sin cambios funcionales: `detect.fps: 5`, grabación **solo eventos** (para comparar cambiando una sola variable).
- PV nuevo `frigate-data-pc1` (HDD de pc1-node). Datos copiados desde rp2 con Frigate detenido (1270 archivos, incluida `frigate.db` → historial de eventos conservado).
- Límites del pod: 4 cores / 4 GiB (antes 3 / 3 GiB en rp2).
- Alerta `InferenciaLenta` baja de 100 ms a **60 ms** (≈ 3× lo normal con OpenVINO).

## Consecuencias

**Positivo:**
- Inferencia ~3× más rápida dentro de Frigate, sin frames saltados.
- rp2 libera hasta 3 cores y el SSD queda para Prometheus.
- 590 G de HDD disponibles si se quiere grabación continua.

**Negativo:**
- El detector usa más CPU cuando hay actividad (~0,75 core a ~9 detecciones/s): OpenVINO paraleliza cada inferencia en varios hilos. En reposo baja. pc1-node tiene 8 hilos, sobra margen.
- **Frigate queda atado a x86:** la configuración `openvino` no funciona en la imagen arm64. Volver a una Pi requiere cambiar el detector a `cpu`.
- Mayor consumo eléctrico que en la Pi.

## Rollback

Mientras exista el PVC `frigate-data` (rp2):
1. `nodeSelector: rp2-node`, `claimName: frigate-data`, límites 3 / 3 GiB
2. Detector `cpu` con `num_threads: 3` y sin bloque `model:`
3. Los eventos ocurridos en pc1-node no estarán en la DB de rp2 (copiar de vuelta si importan)

## Adenda (2026-11-23): perfil AppArmor propio `k8s-frigate`

**Problema:** horas después de migrar, la Tapo se cortó (`i/o timeout` en go2rtc) y Frigate quedó "offline" de forma permanente: al intentar matar su ffmpeg colgado, `os.kill` devolvía `PermissionError` (aun siendo root).

**Causa:** en Ubuntu 26.04 (AppArmor 5) los procesos del contenedor quedan con la etiqueta apilada `cri-containerd.apparmor.d//&unconfined`. El perfil por defecto de containerd solo permite `signal` y `ptrace` hacia `peer=cri-containerd.apparmor.d` → el contenedor no puede enviarse señales a sí mismo. Evidencia en `journalctl -k`: `operation="signal" ... denied_mask="send"` (go2rtc, tempio) y ~3000 `operation="ptrace"` (psutil de Frigate leyendo /proc de ffmpeg); el SIGTERM de Frigate quedó entre los ~300 mensajes suprimidos por rate limit. En las Pi (Ubuntu 25.10) no ocurre.

**Opciones evaluadas:** `appArmorProfile: Unconfined` (rápido, pero quita el confinamiento) · **perfil propio** · volver Frigate a rp2.

**Decisión:** perfil `k8s-frigate` = copia del de containerd que además acepta `peer="k8s-frigate//&unconfined"` en las reglas `signal` y `ptrace`. Se instala con `homelab-ansible/playbooks/apparmor.yml` (en `/etc/apparmor.d/`, se recarga en cada boot) y el deployment lo usa con `appArmorProfile: {type: Localhost, localhostProfile: k8s-frigate}`.

**Verificado:** procesos con `k8s-frigate//&unconfined (enforce)`; `os.kill(SIGTERM)` al ffmpeg de detección funciona y Frigate lo relanza solo en ~10 s; 0 rechazos AppArmor.

**Consecuencia:** si el pod se programa en un nodo sin el perfil cargado → `CreateContainerError`. Frigate está fijado a pc1-node; si se mueve, correr `apparmor.yml` en el nodo nuevo. Otros pods en pc1-node (p. ej. el speaker de MetalLB, también `//&unconfined`) podrían tener el mismo problema si necesitan señales internas.
