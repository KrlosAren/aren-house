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

## Convenciones

- Los archivos se numeran con prefijo (`00-`, `01-`, ...) para controlar el orden de aplicación
- Cada app vive en su propio namespace
- Los Ingress usan el dominio `*.k8s.homelab.local`
