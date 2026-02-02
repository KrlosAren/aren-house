# 10. Almacenamiento de K3s/Containerd en Discos Locales

- **Estado**: Aceptado
- **Fecha**: 2026-01-23

## Contexto

La arquitectura del homelab se basa en que los nodos (`rp2-node`, `rp3-node`) arrancan su sistema de archivos raíz desde un recurso compartido NFS alojado en el gateway (`rp1-master`). Este enfoque centraliza la gestión del sistema operativo.

k3s, nuestra distribución de Kubernetes elegida, utiliza `containerd` como su Container Runtime Interface (CRI). `containerd` es responsable de gestionar las imágenes y el ciclo de vida de los contenedores. Al igual que Docker, `containerd` utiliza drivers de almacenamiento para manejar las capas de las imágenes. El driver por defecto y más eficiente es `overlay2`, que depende de `overlayfs`.

Una limitación fundamental del kernel de Linux es que `overlayfs` no se puede utilizar sobre un sistema de archivos respaldado por NFS. Cuando k3s/containerd se instala en nuestros nodos de arranque por red, su directorio de datos (`/var/lib/rancher`) reside en NFS. En consecuencia, `containerd` no puede usar `overlay2` y recurre al driver de almacenamiento `vfs`.

El driver `vfs` es altamente ineficiente:
- **Sin capas compartidas:** No comparte las capas de las imágenes. Por cada contenedor iniciado, realiza una copia completa de toda la imagen.
- **Alta latencia:** Cada operación de archivo se envía a través de la red al servidor NFS, lo que resulta en tiempos de descarga de imágenes y de inicio de contenedores extremadamente lentos.
- **Alto uso de disco:** La falta de capas compartidas conduce a un consumo exponencial de espacio en disco en el servidor NFS.

Este problema es idéntico al que se encontró previamente con Docker, como se documenta en [ADR-007](./007-docker-storage-overlay.md).

## Decisión

No almacenaremos el directorio de datos de `containerd` en el sistema de archivos raíz respaldado por NFS.

En su lugar, adoptaremos un enfoque de almacenamiento híbrido:
1.  **Utilizar almacenamiento local:** Cada nodo utilizará su dispositivo de almacenamiento local dedicado (`microSD` en `rp2`, `SSD` en `rp3`) para los datos del runtime de contenedores.
2.  **Crear un directorio dedicado:** Se creará un directorio en el punto de montaje del disco local (p. ej., `/var/lib/rancher-local`).
3.  **Usar un enlace simbólico:** La ruta de datos por defecto de `containerd` (`/var/lib/rancher`) será un enlace simbólico que apunte al directorio en el disco local.

Esta configuración está automatizada por el playbook `k3s.yml`.

## Consecuencias

### Positivas
- **Rendimiento:** `containerd` puede utilizar el driver `overlay2`, lo que conduce a descargas de imágenes rápidas y velocidades de inicio de contenedores casi nativas.
- **Eficiencia:** Las capas de las imágenes se comparten entre contenedores, reduciendo significativamente el consumo de espacio en disco.
- **Consistencia:** Esta solución es consistente con el enfoque que ya ha demostrado ser exitoso para el almacenamiento de Docker en este entorno.
- **SO Centralizado:** Conservamos los beneficios de un sistema operativo gestionado de forma centralizada a través de netboot, mientras aislamos las operaciones intensivas de E/S de los contenedores en los discos locales.

### Negativas
- **"Statelessness" reducido:** Los nodos ya no son puramente "stateless", ya que los datos críticos del runtime residen en sus discos locales. Un fallo del disco local afectará al nodo de Kubernetes.
- **Complejidad de configuración:** Cada nodo requiere un disco local presente y montado correctamente. Esto añade un paso al proceso de aprovisionamiento de nodos, aunque está automatizado a través de Ansible (`local-storage.yml` y `k3s.yml`).
