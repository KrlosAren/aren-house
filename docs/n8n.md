# n8n

## Acceso

`http://n8n.k8s.homelab.local`

## Arquitectura

```
┌────────────┐     ┌────────────┐     ┌─────────────┐     ┌────────────────┐
│  Ingress   │────▶│  Service   │────▶│  Deployment │────▶│  PostgreSQL    │
│  n8n.k8s   │     │  n8n-web   │     │  n8n        │     │  10.0.0.1:5432 │
│  .homelab  │     │  :80→5678  │     │  (1 replica)│     │  (Docker)      │
│  .local    │     └────────────┘     └──────┬──────┘     └────────────────┘
└────────────┘                               │
                                             ▼
                                     ┌──────────────┐
                                     │  PVC 5Gi     │
                                     │  (Longhorn)  │
                                     └──────────────┘
```

## Manifiestos

```
k8s-apps/n8n/
├── 00-namespace.yml      # Namespace: n8n-system
├── 01-pg-service.yml     # Service "postgres" (sin selector)
├── 02-pg-endpoints.yml   # EndpointSlice → 10.0.0.1:5432
├── 03-n8n-deployment.yml # Deployment n8n
├── 04-n8n-pvc.yml        # PVC 5Gi (Longhorn)
├── 05-n8n-service.yml    # Service "n8n-web" :80 → :5678
├── 06-n8n-ingress.yml    # Ingress: n8n.k8s.homelab.local
└── secrets/
    └── 00-secrets.yml    # DB_POSTGRESDB_PASSWORD (SOPS+age)
```

### Aplicar

```bash
kubectl apply -f k8s-apps/n8n/
# Los secrets SOPS necesitan: sops -d secrets/00-secrets.yml | kubectl apply -f -
```

## Detalles técnicos

### Problema N8N_PORT

El Service se llama `n8n-web` en vez de `n8n`. Esto es intencional:

Kubernetes inyecta variables de entorno para cada Service en el namespace, siguiendo el patrón `<SERVICE_NAME>_PORT`. Si el Service se llamara `n8n`, k8s inyectaría `N8N_PORT=tcp://10.43.x.x:80`, lo cual **rompe n8n** porque la aplicación espera que `N8N_PORT` sea un número (e.g., `5678`), no una URL.

Solución: nombrar el Service `n8n-web` para que la variable inyectada sea `N8N_WEB_PORT` y no colisione.

### securityContext

```yaml
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

n8n necesita escribir en `/home/node/.n8n` (montado como PVC). Sin `fsGroup: 1000`, el volumen se monta como root y n8n no puede escribir.

### Conexión a PostgreSQL

n8n se conecta a PostgreSQL externo (Docker en rp1-master) via el patrón Service sin selector + EndpointSlice. Ver [docs/databases.md](databases.md) para más detalles.

Variables de entorno:
```yaml
DB_TYPE: postgresdb
DB_POSTGRESDB_HOST: postgres      # Nombre del Service
DB_POSTGRESDB_PORT: "5432"
DB_POSTGRESDB_DATABASE: n8n
DB_POSTGRESDB_USER: admin
DB_POSTGRESDB_PASSWORD: <secret>  # Desde SOPS secret
```

### Storage

El PVC usa `storageClassName: longhorn` (5Gi). Es la única app del cluster que usa Longhorn en vez de local-path.

## Troubleshooting

| Problema | Causa | Solución |
|----------|-------|----------|
| n8n no arranca, error `N8N_PORT` | Service nombrado `n8n` inyecta variable conflictiva | Renombrar Service a `n8n-web` |
| Permission denied en `/home/node/.n8n` | PVC montado como root | Agregar `fsGroup: 1000` en securityContext |
| `ECONNREFUSED` a postgres | EndpointSlice no apunta a IP correcta | Verificar que `02-pg-endpoints.yml` tiene `10.0.0.1` |
| Pod en `Pending` | PVC no se provisiona | Verificar que Longhorn está funcionando |
