# Bases de Datos

## Arquitectura

```
┌─────────────────────────────────────────────────┐
│                   Cluster k3s                    │
│                                                  │
│  ┌──────────┐     ┌───────────┐     ┌────────┐ │
│  │   n8n    │────▶│  Service  │────▶│Endpoint│ │
│  │   Pod    │     │  postgres │     │ Slice  │ │
│  └──────────┘     │  (no sel) │     │10.0.0.1│ │
│                   └───────────┘     └────┬───┘ │
│                                          │      │
└──────────────────────────────────────────┼──────┘
                                           │
                                           ▼
                              ┌─────────────────────┐
                              │   rp1-master         │
                              │   Docker + systemd   │
                              │   PostgreSQL 18      │
                              │   :5432              │
                              └─────────────────────┘
```

PostgreSQL corre **fuera del cluster**, en Docker en rp1-master, gestionado por systemd. Los pods del cluster acceden a él via un Service sin selector + EndpointSlice.

Ver [ADR-013](decisions/013-databases-outside-k8s.md) para la decisión arquitectónica.

## PostgreSQL

### Stack Docker

**Ubicación**: `stacks/storage-apps/`

```
stacks/storage-apps/
├── docker-compose.yml          # PostgreSQL 18-alpine
├── .env                        # Credenciales
└── homelab-postgres.service    # Servicio systemd
```

### Configuración

- **Imagen**: `postgres:18-alpine`
- **Puerto**: `5432` (expuesto al host)
- **Datos**: `/backup/data/postgres` (SSD 500GB de rp1-master)
- **Usuario**: `admin`
- **Base de datos**: `homelab`

### Servicio systemd

El archivo `homelab-postgres.service` se instala en rp1-master para que PostgreSQL arranque automáticamente con el sistema:

```bash
# Copiar el service file
sudo cp homelab-postgres.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable homelab-postgres
sudo systemctl start homelab-postgres

# Verificar
sudo systemctl status homelab-postgres
```

### Comandos útiles

```bash
# Conectar al PostgreSQL
docker exec -it postgres psql -U admin -d homelab

# Ver bases de datos
docker exec -it postgres psql -U admin -d homelab -c '\l'

# Backup
docker exec postgres pg_dump -U admin homelab > backup.sql
```

## Exponer servicios externos a k8s

### Patrón: Service sin selector + EndpointSlice

Cuando un servicio corre fuera del cluster (como PostgreSQL en Docker), se puede exponer dentro de k8s usando un Service sin selector y un EndpointSlice manual:

```yaml
# Service sin selector
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: mi-namespace
spec:
  ports:
    - port: 5432
      targetPort: 5432
  # Sin selector - no apunta a pods
---
# EndpointSlice apuntando al host externo
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: postgres
  namespace: mi-namespace
  labels:
    kubernetes.io/service-name: postgres
addressType: IPv4
ports:
  - port: 5432
    protocol: TCP
endpoints:
  - addresses:
      - "10.0.0.1"  # IP del host donde corre PostgreSQL
```

Los pods del namespace pueden entonces conectarse a `postgres:5432` como si fuera un servicio interno.

### Ejemplo real: n8n → PostgreSQL

```
k8s-apps/n8n/
├── 01-pg-service.yml      # Service "postgres" sin selector
├── 02-pg-endpoints.yml    # EndpointSlice → 10.0.0.1:5432
└── 03-n8n-deployment.yml  # DB_POSTGRESDB_HOST=postgres
```

n8n se conecta a `postgres:5432` (nombre DNS del Service), que el EndpointSlice redirige a `10.0.0.1:5432` (rp1-master).
