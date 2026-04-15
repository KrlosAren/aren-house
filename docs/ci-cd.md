# CI/CD y GitHub Actions

## Arquitectura

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────────┐     ┌─────────────────────────┐
│  git push    │────▶│  security-gate  │────▶│  detect-changes  │────▶│  build-<app>            │
│  (main)      │     │  (ubuntu-latest)│     │  (ubuntu-latest) │     │  (self-hosted, homelab) │
└──────────────┘     └─────────────────┘     └──────────────────┘     └──────────┬──────────────┘
                                                                                  │
                                                                                  ▼
                                                                     ┌─────────────────────────┐
                                                                     │  registry.k8s.homelab    │
                                                                     │  .local/test-app:latest  │
                                                                     └─────────────────────────┘
```

El pipeline se ejecuta en dos tipos de runners:
- **ubuntu-latest (público)**: Jobs de verificación (security-gate, detect-changes)
- **self-hosted, homelab (rp1-master)**: Jobs de build que acceden al registry local

## Self-hosted Runner

### Instalación

El runner se instala en rp1-master via Ansible:

```bash
cd homelab-ansible
ansible-playbook playbooks/github-runner.yml -e "github_token=TU_TOKEN"
```

**Configuración del runner:**
- **Versión**: 2.331.0 (arm64)
- **Directorio**: `/home/admin/actions-runner`
- **Labels**: `homelab`, `self-hosted`, `arm64`
- **Usuario**: `admin`
- **Servicio**: systemd (auto-start)

Para obtener el token: GitHub repo → Settings → Actions → Runners → New self-hosted runner → copiar token.

### Verificar estado

```bash
# En rp1-master
cd /home/admin/actions-runner
./svc.sh status
```

## Protecciones para repos públicos

El repositorio es público, lo que significa que PRs de forks podrían ejecutar código en el self-hosted runner. El job `security-gate` previene esto:

```yaml
security-gate:
  runs-on: ubuntu-latest  # Corre en runner público, NO en self-hosted
  steps:
    - Verifica que el repo sea KrlosAren/aren-house
    - Verifica que el evento NO sea pull_request ni pull_request_target
    - Verifica que el actor sea KrlosAren
```

Solo si las 3 condiciones se cumplen, el output `safe=true` permite que los jobs siguientes se ejecuten en el self-hosted runner.

## Workflow: Build and Deploy

**Archivo**: `.github/workflows/test-app.yml`

**Triggers**:
- Push a `main` que modifique archivos en `apps/`
- `workflow_dispatch` manual (con input `app` opcional)

### Jobs

#### 1. security-gate
Verifica que el trigger es seguro (ver sección anterior).

#### 2. detect-changes
Compara `HEAD~1` con `HEAD` para detectar qué apps cambiaron. También acepta input manual via `workflow_dispatch`.

#### 3. build-test-app
Solo se ejecuta si `detect-changes` detectó cambios en `apps/test-app/`. Construye la imagen Docker y la pushea al registry local:

```bash
# Tags generados:
registry.k8s.homelab.local/test-app:<sha-completo>
registry.k8s.homelab.local/test-app:<sha-corto>
registry.k8s.homelab.local/test-app:latest
```

**Labels OCI incluidas**: `org.opencontainers.image.revision`, `org.opencontainers.image.created`

#### 4. notify-failure
Se ejecuta si algún job falla. Actualmente imprime info básica (placeholder para Slack/Discord).

## Estructura de archivos

```
apps/
└── test-app/           # Código fuente de la aplicación (Express/Node.js)
    ├── Dockerfile
    ├── package.json
    └── ...

k8s-apps/test-app/     # Manifiestos k8s para desplegar la app
├── 00-namespace.yml    # Namespace: simple-app
├── 01-deployment.yml   # 2 replicas, imagen del registry local
├── 02-service.yml      # Port 3000
└── 04-ingress.yml      # test-app.k8s.homelab.local

.github/workflows/
└── test-app.yml        # Pipeline CI/CD
```

## Agregar una nueva app al pipeline

1. Crear el código fuente en `apps/<nueva-app>/` con un `Dockerfile`

2. En `.github/workflows/test-app.yml`, agregar detección en el job `detect-changes`:
   ```yaml
   - name: Check nueva-app changes
     id: nueva-app
     run: |
       if echo "$CHANGED" | grep -q "^apps/nueva-app/"; then
         echo "changed=true" >> $GITHUB_OUTPUT
       fi
   ```

3. Agregar el output en el job `detect-changes`:
   ```yaml
   outputs:
     nueva-app: ${{ steps.nueva-app.outputs.changed }}
   ```

4. Crear un nuevo job `build-nueva-app` copiando la estructura de `build-test-app`, cambiando `IMAGE_NAME` y el path del build context.

5. Crear manifiestos k8s en `k8s-apps/<nueva-app>/`

## Deploy (pendiente)

Actualmente el deploy es manual (`kubectl apply`). El workflow tiene un job de deploy comentado como placeholder para futura integración con ArgoCD o `kubectl set image`.
