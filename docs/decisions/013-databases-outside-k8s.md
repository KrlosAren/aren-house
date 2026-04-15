# ADR-013: Bases de datos fuera del cluster k8s

## Estado

Aceptado (2026-02)

## Contexto

Al migrar n8n al cluster k3s, se necesitaba decidir dónde correr PostgreSQL:
1. Dentro del cluster como StatefulSet
2. Fuera del cluster en Docker+systemd en rp1-master

## Decisión

PostgreSQL corre en Docker en rp1-master, gestionado por systemd. Los pods del cluster acceden a él via Service sin selector + EndpointSlice.

## Razones

- **Simplicidad de backups**: Los datos están en `/backup/data/postgres` en el SSD de 500GB de rp1-master, fácil de respaldar con herramientas estándar (`pg_dump`, rsync)
- **Menor consumo de recursos**: No se necesita un StatefulSet con PVCs replicados, operadores, o lógica de failover en un cluster de 3 nodos
- **Storage confiable**: El SSD de 500GB de rp1-master es el disco más grande y rápido del homelab. Dentro del cluster, los datos irían a rp3 (240GB) o local-path sin replicación
- **Control directo**: `docker exec` para acceder a psql, logs directos, systemd para auto-start

## Consecuencias

- Las apps k8s que necesiten PostgreSQL deben crear un Service sin selector + EndpointSlice apuntando a `10.0.0.1:5432`
- Si rp1-master se cae, tanto el cluster como la base de datos dejan de funcionar (aceptable porque rp1-master es el control plane)
- Si en el futuro se necesita alta disponibilidad de la DB, se puede migrar a un operador k8s (e.g., CloudNativePG)

## Alternativas consideradas

- **PostgreSQL como StatefulSet**: Más complejo, requiere operador para backups/failover, consume recursos del cluster
- **SQLite embebido**: Más simple pero no soporta acceso concurrente ni múltiples apps
