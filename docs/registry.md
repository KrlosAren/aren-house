# Registry Privado

## Arquitectura

```
┌──────────────┐     ┌──────────────────────┐     ┌─────────────────┐
│  docker push │────▶│  registry.k8s        │────▶│  PVC 20Gi       │
│  (CI/CD)     │     │  .homelab.local:80   │     │  (local-path)   │
└──────────────┘     └──────────────────────┘     └─────────────────┘
                              ▲
                              │ HTTP (insecure)
┌──────────────┐     ┌───────┴──────────────┐
│  containerd  │────▶│  registries.yaml     │
│  (k3s nodes) │     │  mirror → HTTP       │
└──────────────┘     └──────────────────────┘
```

El registry corre dentro del cluster k8s, expuesto via Ingress. Los nodos lo acceden por HTTP (no HTTPS) usando un mirror configurado en containerd.

## URLs

| URL | Servicio |
|-----|----------|
| `registry.k8s.homelab.local` | Docker Registry v2 |
| `registry-ui.k8s.homelab.local` | Registry UI (web) |

## Manifiestos

```
k8s-apps/registry/
├── 00-namespace.yml              # Namespace: registry
├── 01-registry-pvc.yml           # PVC 20Gi (local-path)
├── 02-registry-deployment.yml    # Registry 2.8.1
├── 03-registry-ui-deployment.yml # UI web (joxit/docker-registry-ui)
├── 04-registry-service.yml       # Service registry:5000
├── 05-registry-ui-service.yml    # Service registry-ui:80
├── 06-ingress.yml                # Ingress para ambos hosts
└── secrets/
    └── 07-registry-secret.yml    # REGISTRY_HTTP_SECRET (SOPS+age)
```

### Aplicar

```bash
kubectl apply -f k8s-apps/registry/
# Los secrets SOPS necesitan: sops -d secrets/07-registry-secret.yml | kubectl apply -f -
```

## Configuración en los nodos

Para que containerd (k3s) pueda hacer pull del registry local por HTTP, cada nodo necesita dos cosas:

### 1. `/etc/rancher/k3s/registries.yaml`

```yaml
mirrors:
  registry.k8s.homelab.local:
    endpoint:
      - "http://registry.k8s.homelab.local"
  docker.io:
    endpoint:
      - "https://registry-1.docker.io"
```

Esto le dice a containerd que use HTTP para el registry local.

### 2. `/etc/hosts`

```
10.0.0.50 registry.k8s.homelab.local
```

Containerd no usa dnsmasq del sistema para resolver nombres, necesita `/etc/hosts` o un DNS que conozca el dominio.

### Automatización

El playbook `registry.yml` configura ambos archivos y reinicia k3s:

```bash
cd homelab-ansible
ansible-playbook playbooks/registry.yml
```

Este playbook también configura Docker (`insecure-registries`) si Docker está instalado en el nodo.

## Flujo de push (CI/CD)

```
1. GitHub Actions detecta cambios en apps/test-app/
2. Self-hosted runner (rp1-master) ejecuta docker build
3. docker push registry.k8s.homelab.local/test-app:latest
4. La imagen llega al registry via Ingress (HTTP)
```

## Flujo de pull (desde un pod)

```
1. Pod spec: image: registry.k8s.homelab.local/test-app:latest
2. containerd lee registries.yaml → usa HTTP mirror
3. Resuelve registry.k8s.homelab.local via /etc/hosts → 10.0.0.50
4. HTTP GET → MetalLB → Traefik k3s → Ingress → registry Service → Pod registry
5. Imagen descargada al nodo
```

## Pushear una imagen manualmente

```bash
# Desde rp1-master (o cualquier máquina con Docker configurado)
docker build -t registry.k8s.homelab.local/mi-app:latest .
docker push registry.k8s.homelab.local/mi-app:latest

# Verificar
curl http://registry.k8s.homelab.local/v2/_catalog
```

## Registry UI

Accede a `http://registry-ui.k8s.homelab.local` para ver las imágenes almacenadas, tags, y borrar imágenes (`DELETE_IMAGES=true`).

## Troubleshooting

| Problema | Causa | Solución |
|----------|-------|----------|
| `pull: connection refused` | containerd no encuentra el registry | Verificar `/etc/hosts` y `registries.yaml` en el nodo |
| `pull: http: server gave HTTP response to HTTPS client` | containerd intenta HTTPS | Verificar mirror HTTP en `registries.yaml`, reiniciar k3s |
| `push: connection refused` | Docker no tiene insecure-registries | Agregar `insecure-registries` en `/etc/docker/daemon.json` |
| UI muestra CORS error | CORS headers no configurados | Verificar env `REGISTRY_HTTP_HEADERS_Access-Control-*` en deployment |
