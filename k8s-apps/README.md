# Aplicaciones Kubernetes (k8s-apps)

Manifiestos de Kubernetes para el cluster k3s del homelab. Cada directorio contiene los manifiestos YAML de una aplicación, numerados para aplicarse en orden.

## Arquitectura del Ingress

El cluster usa **MetalLB + Traefik** para exponer servicios:

```
Cliente (browser/curl)
    │
    ▼
DNS (dnsmasq)
    │  *.k8s.homelab.local → 10.0.0.50
    ▼
MetalLB (L2 mode)
    │  Asigna IP 10.0.0.50 al Service tipo LoadBalancer
    ▼
Traefik (Ingress Controller, incluido en k3s)
    │  Lee los recursos Ingress y rutea por hostname
    ▼
Service → Pod
```

Para exponer un servicio en el cluster:
1. Crear un `Service` (ClusterIP) que apunte a tus pods
2. Crear un `Ingress` con el hostname deseado (ej: `miapp.k8s.homelab.local`)
3. El DNS ya resuelve `*.k8s.homelab.local` a `10.0.0.50` (configurado en dnsmasq)

## Storage

Actualmente usamos **local-path** (provisioner incluido en k3s) para PersistentVolumeClaims. Los workloads con necesidades de I/O (como Prometheus) se fuerzan a nodos con SSD via `nodeSelector`.

> Longhorn (storage distribuido) está evaluado pero no implementado. Se activará cuando se necesite replicación de datos entre nodos.

## Aplicaciones

| Directorio | Descripción | Namespace | Estado |
|------------|-------------|-----------|--------|
| `metallb/` | Configuración del pool de IPs para LoadBalancer | metallb-system | Activo |
| `monitoring-stack/` | Prometheus con service discovery de k8s | monitoring | Activo |
| `registry/` | Registry privado de imágenes + UI web | registry | Activo |
| `storage-learning/` | App de prueba nginx con PVC (ejercicio de aprendizaje) | storage-lab | Ejercicio |
| `storage-longhorn/` | Manifiesto de Longhorn (no aplicado) | longhorn-system | No aplicado |

## Uso

```bash
# Aplicar una app completa (los archivos se aplican en orden alfabético)
kubectl apply -f k8s-apps/monitoring-stack/

# Verificar estado
kubectl get all -n monitoring

# Eliminar una app
kubectl delete -f k8s-apps/monitoring-stack/
```

## Secrets con SOPS + age

Los secrets se gestionan con [SOPS](https://github.com/getsops/sops) + [age](https://github.com/FiloSottile/age). Los valores se encriptan antes de commitear a git.

### Setup (una sola vez)

```bash
# Instalar herramientas
brew install sops age

# Generar llaves (ya hecho, guardada en ~/.sops/age-key.txt)
age-keygen -o ~/.sops/age-key.txt

# Configurar variable de entorno (agregar a ~/.zshrc)
export SOPS_AGE_KEY_FILE=~/.sops/age-key.txt
```

La llave pública está en `.sops.yaml` (va a git). La llave privada está en `~/.sops/age-key.txt` (NO va a git).

### Convención de archivos

Los archivos encriptados van en una subcarpeta `secrets/` dentro de cada app:

```
k8s-apps/mi-app/
├── 00-namespace.yml
├── 01-deployment.yml
├── 02-service.yml
└── secrets/
    └── 01-secret.yml    ← encriptado con SOPS
```

Esto permite hacer `kubectl apply -f k8s-apps/mi-app/` sin que falle por archivos encriptados.

### Deploy de una app con secrets

```bash
# 1. Primero los secrets (desencriptar y aplicar)
sops decrypt k8s-apps/mi-app/secrets/01-secret.yml | kubectl apply -f -

# 2. Luego el resto de manifiestos
kubectl apply -f k8s-apps/mi-app/
```

### Operaciones comunes

```bash
# Crear un secret nuevo
cat > k8s-apps/mi-app/secrets/01-secret.yml << 'EOF'
apiVersion: v1
kind: Secret
metadata:
  name: mi-secret
  namespace: mi-app
type: Opaque
stringData:
  MI_PASSWORD: "valor-real-aqui"
EOF
sops encrypt -i k8s-apps/mi-app/secrets/01-secret.yml

# Editar un secret existente (abre editor con valores en claro)
sops k8s-apps/mi-app/secrets/01-secret.yml

# Ver un secret desencriptado (sin aplicar)
sops decrypt k8s-apps/mi-app/secrets/01-secret.yml

# Aplicar al cluster
sops decrypt k8s-apps/mi-app/secrets/01-secret.yml | kubectl apply -f -
```

### Cómo se referencia en un Deployment

```yaml
env:
  - name: MI_PASSWORD
    valueFrom:
      secretKeyRef:
        name: mi-secret      # nombre del Secret
        key: MI_PASSWORD      # key dentro del Secret
```

## Convenciones

- Los archivos se numeran con prefijo (`00-`, `01-`, ...) para controlar el orden de aplicación
- Cada app vive en su propio namespace
- Los Ingress usan el dominio `*.k8s.homelab.local`
- Los secrets encriptados van en subcarpeta `secrets/` (ver sección SOPS)
