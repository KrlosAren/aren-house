# ADR-016: local-path sobre Longhorn para storage stateful

**Estado:** Aceptado  
**Fecha:** 2026-04-20

## Contexto

El cluster k3s tiene tres apps stateful con PVCs en Longhorn:
- Prometheus (20Gi, namespace monitoring)
- Grafana (5Gi, namespace monitoring)
- n8n (5Gi, namespace n8n-system)

Longhorn comenzó a causar problemas de estabilidad: el nodo rp3 quedó bloqueado con errores de kernel (`rcu_preempt detected expedited stalls`) relacionados con I/O de iSCSI. Además, Longhorn requiere dos capas de instalación (Ansible + Helm), dependencias de OS (iSCSI, open-iscsi), y tiene overhead de replicación de red que en un homelab de 3 nodos no aporta valor suficiente.

## Decisión

Migrar todos los PVCs de `storageClassName: longhorn` a `storageClassName: local-path`.

Distribución manual de nodos (nodeAffinity):
- **rp2-node**: Prometheus — app I/O intensiva con TSDB de 20Gi
- **rp3-node**: Grafana y n8n — apps más livianas

## Consecuencias

**Positivo:**
- Elimina la capa iSCSI que causaba bloqueos de kernel en rp3
- Simplifica la arquitectura: no se necesita Longhorn instalado ni sus dependencias de OS
- local-path es nativo de k3s, sin componentes adicionales
- Menos overhead de CPU/red en los nodos

**Negativo:**
- Sin replicación: si un nodo falla, los datos de las apps en ese nodo no están disponibles hasta que el nodo vuelva
- Los pods quedan pinados a nodos específicos via nodeAffinity — si el nodo muere, hay que intervención manual para mover los datos

**Aceptable para homelab porque:**
- Prometheus tiene retención de 15 días y puede reconstruir datos haciendo scraping
- Los dashboards de Grafana están provisionados via ConfigMaps (no se pierden con el volumen)
- n8n usa PostgreSQL para workflows y ejecuciones; el volumen local solo tiene configuración

## Procedimiento de migración

Ver runbook: `docs/runbooks/migrar-longhorn-a-local-path.md`
