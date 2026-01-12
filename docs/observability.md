# Stack de Observabilidad

## Arquitectura
```
┌─────────────────────────────────────────────────────┐
│                     rp2-node                        │
│                                                     │
│  ┌─────────────┐         ┌─────────────┐           │
│  │ Prometheus  │────────►│   Grafana   │           │
│  │   :9090     │         │    :3000    │           │
│  └──────┬──────┘         └─────────────┘           │
│         │                                          │
│         │ scrape cada 15s                          │
└─────────┼──────────────────────────────────────────┘
          │
          ├──────────────┬──────────────┐
          ▼              ▼              ▼
    ┌───────────┐  ┌───────────┐  ┌───────────┐
    │rp1-master │  │ rp2-node  │  │ rp3-node  │
    │node_export│  │node_export│  │node_export│
    │  :9100    │  │  :9100    │  │  :9100    │
    └───────────┘  └───────────┘  └───────────┘
```

---

## Componentes

| Componente | Función | Puerto | Ubicación |
|------------|---------|--------|-----------|
| Prometheus | Recolecta y almacena métricas | 9090 | rp2-node |
| Grafana | Visualización y dashboards | 3000 | rp2-node |
| node_exporter | Expone métricas del sistema | 9100 | Todos los nodos |

---

## Despliegue

### Stack principal (Prometheus + Grafana)

Ubicación: `~/stacks/observability/` en rp2-node
```bash
# Desplegar
cd ~/stacks/observability
docker compose up -d

# Ver logs
docker compose logs -f

# Detener
docker compose down
```

### node_exporter (todos los nodos)
```bash
ansible-playbook playbooks/node-exporter.yml
```

---

## Acceso

| Servicio | URL | Credenciales |
|----------|-----|--------------|
| Prometheus | http://10.0.0.2:9090 | - |
| Grafana | http://10.0.0.2:3000 | admin / admin |

---

## Configuración de Prometheus

Archivo: `~/stacks/observability/prometheus/prometheus.yml`
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node_exporter'
    static_configs:
      - targets:
          - '10.0.0.1:9100'  # Gateway
          - '10.0.0.2:9100'  # rp2
          - '10.0.0.3:9100'  # rp3
```

### Agregar nuevos targets

1. Editar `prometheus/prometheus.yml`
2. Agregar target en `static_configs`
3. Reiniciar Prometheus:
```bash
   docker compose restart prometheus
```

---

## Configuración de Grafana

### Agregar Prometheus como datasource

1. Ir a Configuration → Data Sources
2. Add data source → Prometheus
3. URL: `http://prometheus:9090`
4. Save & Test

### Dashboards recomendados

| Dashboard | ID | Descripción |
|-----------|-----|-------------|
| Node Exporter Full | 1860 | Métricas completas del sistema |
| Docker | 893 | Métricas de contenedores |

Para importar:
1. Ir a Dashboards → Import
2. Ingresar ID
3. Seleccionar datasource Prometheus

---

## Métricas Disponibles

### node_exporter

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

### Prometheus no puede conectar a node_exporter
```bash
# Verificar que node_exporter está corriendo
systemctl status node_exporter

# Verificar puerto
curl http://10.0.0.1:9100/metrics

# Verificar firewall
sudo ufw status | grep 9100
```

### Grafana no muestra datos

1. Verificar datasource en Grafana
2. Verificar que Prometheus tiene datos:
```
   http://10.0.0.2:9090/targets
```
3. Verificar queries en panel
