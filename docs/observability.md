# Stack de Observabilidad

## Estado actual

La observabilidad está en proceso de migración de Docker a Kubernetes (k3s). El objetivo es consolidar todos los servicios de monitoreo dentro del cluster.

| Componente | Estado | Ubicación |
|------------|--------|-----------|
| Prometheus | Migrado a k8s | `k8s-apps/monitoring-stack/` |
| Grafana | Pendiente migrar | Docker en rp2-node (`stacks/observability/`) |
| node_exporter | Activo (systemd) | Todos los nodos |
| Loki | Pendiente | - |
| Alertmanager | Pendiente | - |

## Arquitectura actual

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes (k3s)                          │
│                                                              │
│  ┌─────────────────┐                                        │
│  │ Prometheus (k8s) │  ← namespace: monitoring               │
│  │ rp3-node (SSD)   │  ← nodeSelector: rp3-node             │
│  │ Retención: 15d    │  ← PVC: local-path                    │
│  └────────┬──────────┘                                       │
│           │ scrape via kubernetes_sd_configs                  │
│           │                                                  │
│           ├── kubernetes-apiservers (API server)             │
│           ├── kubernetes-nodes (kubelet)                     │
│           ├── kubernetes-cadvisor (contenedores)             │
│           └── kubernetes-pods (auto-discovery)               │
│                                                              │
│  Acceso: prometheus.k8s.homelab.local (via Ingress)          │
└──────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Docker (legacy)                            │
│                                                              │
│  ┌─────────────┐         ┌─────────────┐                    │
│  │   Grafana   │────────►│ Prometheus  │ (Docker, legacy)    │
│  │   :3000     │         │   :9090     │                     │
│  └─────────────┘         └─────────────┘                     │
│  Ubicación: stacks/observability/ en rp2-node                │
└──────────────────────────────────────────────────────────────┘

┌───────────┐  ┌───────────┐  ┌───────────┐
│rp1-master │  │ rp2-node  │  │ rp3-node  │
│node_export│  │node_export│  │node_export│
│  :9100    │  │  :9100    │  │  :9100    │
└───────────┘  └───────────┘  └───────────┘
```

---

## Prometheus en Kubernetes

Prometheus corre como Deployment en el namespace `monitoring`, forzado al nodo `rp3-node` (SSD 240GB) para buen rendimiento de I/O.

### Manifiestos

Los manifiestos están en `k8s-apps/monitoring-stack/` y se aplican en orden numérico:

| Archivo | Recurso | Descripción |
|---------|---------|-------------|
| `00-namespace.yml` | Namespace | Crea `monitoring` |
| `01-prometheus-rbac.yml` | ServiceAccount, ClusterRole, ClusterRoleBinding | Permisos para descubrir pods/nodos |
| `02-prometheus-config.yml` | ConfigMap | Configuración de scrape jobs |
| `03-prometheus-pvc.yml` | PersistentVolumeClaim | Storage para datos (local-path) |
| `04-prometheus-deployment.yml` | Deployment | Pod de Prometheus |
| `05-prometheus-service.yml` | Service | Expone Prometheus en el cluster |
| `06-prometheus-ingress.yml` | Ingress | Acceso via `prometheus.k8s.homelab.local` |

### Desplegar / Actualizar

```bash
# Aplicar todos los manifiestos
kubectl apply -f k8s-apps/monitoring-stack/

# Ver estado
kubectl get all -n monitoring

# Ver logs
kubectl logs -l app=prometheus -n monitoring -f
```

### Scrape jobs configurados

| Job | Descubre via | Qué monitorea |
|-----|-------------|---------------|
| `kubernetes-apiservers` | endpoints | API server de k3s |
| `kubernetes-nodes` | node | Métricas de kubelet por nodo |
| `kubernetes-cadvisor` | node (/metrics/cadvisor) | Métricas de contenedores |
| `kubernetes-pods` | pod (annotation `prometheus.io/scrape: "true"`) | Auto-discovery de pods |

### Acceso

| Servicio | URL |
|----------|-----|
| Prometheus (k8s) | http://prometheus.k8s.homelab.local |

---

## node_exporter

Instalado como servicio systemd en todos los nodos via Ansible.

### Despliegue
```bash
ansible-playbook playbooks/node-exporter.yml
```

### Verificar
```bash
# Verificar que está corriendo
systemctl status node_exporter

# Verificar métricas
curl http://10.0.0.1:9100/metrics
curl http://10.0.0.2:9100/metrics
curl http://10.0.0.3:9100/metrics
```

### Métricas principales

| Métrica | Descripción |
|---------|-------------|
| `node_cpu_seconds_total` | Uso de CPU |
| `node_memory_MemTotal_bytes` | Memoria total |
| `node_memory_MemAvailable_bytes` | Memoria disponible |
| `node_filesystem_avail_bytes` | Espacio en disco |
| `node_network_receive_bytes_total` | Tráfico de red recibido |
| `node_network_transmit_bytes_total` | Tráfico de red enviado |
| `node_load1` | Load average 1 min |

### Queries útiles en Prometheus
```promql
# Uso de CPU por nodo
100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# Memoria usada en porcentaje
100 * (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)

# Espacio en disco usado
100 - (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"} * 100)

# Tráfico de red
rate(node_network_receive_bytes_total[5m])
```

---

## Grafana (Docker - pendiente migrar)

Actualmente Grafana sigue corriendo en Docker en rp2-node.

### Ubicación
```
stacks/observability/docker-compose.yml
```

### Acceso

| Servicio | URL | Credenciales |
|----------|-----|--------------|
| Grafana | http://10.0.0.2:3000 | admin / (cambiar al primer login) |

### Dashboards recomendados

| Dashboard | ID | Descripción |
|-----------|-----|-------------|
| Node Exporter Full | 1860 | Métricas completas del sistema |
| Docker | 893 | Métricas de contenedores |

Para importar: Dashboards > Import > Ingresar ID > Seleccionar datasource Prometheus.

---

## Alertas (futuro)

Prometheus puede enviar alertas via Alertmanager. Configuración básica:
```yaml
# En prometheus.yml
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']

rule_files:
  - 'alerts/*.yml'
```

Ejemplo de regla de alerta:
```yaml
# alerts/node.yml
groups:
  - name: node
    rules:
      - alert: HighMemoryUsage
        expr: (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Alta uso de memoria en {{ $labels.instance }}"
```

---

## Troubleshooting

### Prometheus no puede conectar a targets

```bash
# Ver targets y su estado
# Abrir http://prometheus.k8s.homelab.local/targets

# Ver logs del pod
kubectl logs -l app=prometheus -n monitoring

# Verificar RBAC
kubectl get clusterrolebinding prometheus -o yaml
```

### Prometheus no tiene datos

1. Verificar que el pod está running: `kubectl get pods -n monitoring`
2. Verificar PVC: `kubectl get pvc -n monitoring`
3. Verificar targets en la UI de Prometheus

### Grafana no muestra datos

1. Verificar datasource en Grafana (debe apuntar a Prometheus)
2. Verificar que Prometheus tiene datos en sus targets
3. Verificar queries en panel

---

## Plan de migración

### Completado
- [x] Prometheus migrado a k8s con service discovery nativo
- [x] Ingress configurado para acceso via `prometheus.k8s.homelab.local`

### Pendiente
- [ ] Migrar Grafana al cluster k8s
- [ ] Agregar Loki para logs centralizados
- [ ] Configurar Alertmanager con notificaciones
- [ ] Dashboard específico de k3s (pods, deployments, services)
- [ ] Configurar retención de métricas adecuada
