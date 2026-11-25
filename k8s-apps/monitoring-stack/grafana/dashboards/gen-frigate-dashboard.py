# Genera 10-grafana-dashboard-frigate.yml:
#   python3 k8s-apps/monitoring-stack/grafana/dashboards/gen-frigate-dashboard.py k8s-apps/monitoring-stack/grafana/10-grafana-dashboard-frigate.yml
import json, sys
DS = {"type": "prometheus", "uid": "${datasource}"}
pid = [0]
def nid():
    pid[0] += 1; return pid[0]
def tgt(expr, legend="", ref="A", instant=False):
    t = {"datasource": DS, "expr": expr, "legendFormat": legend, "refId": ref}
    if instant: t["instant"] = True
    return t
def row(title, y):
    return {"type": "row", "title": title, "collapsed": False, "id": nid(), "panels": [],
            "gridPos": {"h": 1, "w": 24, "x": 0, "y": y}}
def stat(title, expr, x, y, w=4, unit="none", dec=1, thresholds=None, desc=""):
    th = thresholds or [{"color": "green", "value": None}]
    return {"type": "stat", "title": title, "description": desc, "id": nid(), "datasource": DS,
            "gridPos": {"h": 4, "w": w, "x": x, "y": y},
            "targets": [tgt(expr, instant=True)],
            "fieldConfig": {"defaults": {"unit": unit, "decimals": dec,
                            "thresholds": {"mode": "absolute", "steps": th}}, "overrides": []},
            "options": {"colorMode": "background", "graphMode": "none", "justifyMode": "center",
                        "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
                        "textMode": "value"}}
def ts(title, targets, x, y, w=12, h=8, unit="none", desc="", stack=False, bars=False, minv=None, maxv=None):
    custom = {"drawStyle": "bars" if bars else "line", "fillOpacity": 60 if bars else 15,
              "lineWidth": 2, "showPoints": "never", "spanNulls": True,
              "stacking": {"mode": "normal" if stack else "none", "group": "A"}}
    d = {"unit": unit, "custom": custom}
    if minv is not None: d["min"] = minv
    if maxv is not None: d["max"] = maxv
    return {"type": "timeseries", "title": title, "description": desc, "id": nid(), "datasource": DS,
            "gridPos": {"h": h, "w": w, "x": x, "y": y}, "targets": targets,
            "fieldConfig": {"defaults": d, "overrides": []},
            "options": {"legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
                        "tooltip": {"mode": "multi", "sort": "desc"}}}
def gauge(title, expr, x, y, w=6, desc="", warn=70, crit=90):
    return {"type": "gauge", "title": title, "description": desc, "id": nid(), "datasource": DS,
            "gridPos": {"h": 6, "w": w, "x": x, "y": y}, "targets": [tgt(expr, instant=True)],
            "fieldConfig": {"defaults": {"unit": "percent", "min": 0, "max": 100, "decimals": 1,
                "thresholds": {"mode": "absolute", "steps": [
                    {"color": "green", "value": None}, {"color": "orange", "value": warn},
                    {"color": "red", "value": crit}]}}, "overrides": []},
            "options": {"reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
                        "showThresholdLabels": False, "showThresholdMarkers": True}}

CAM = 'camera_name="$camera"'
P = []
# --- Estado
P.append(row("Estado de la cámara", 0))
P += [
 stat("FPS cámara", f'frigate_camera_fps{{{CAM}}}', 0, 1, desc="Frames por segundo recibidos (configurado: 5)",
      thresholds=[{"color": "red", "value": None}, {"color": "orange", "value": 3}, {"color": "green", "value": 4.5}]),
 stat("FPS detección", f'frigate_detection_fps{{{CAM}}}', 4, 1, desc="Detecciones por segundo; 0 = escena sin movimiento (normal)"),
 stat("Frames saltados", f'frigate_skipped_fps{{{CAM}}}', 8, 1, desc="> 0 = la CPU no da abasto",
      thresholds=[{"color": "green", "value": None}, {"color": "orange", "value": 0.1}, {"color": "red", "value": 1}]),
 stat("Inferencia", 'max(frigate_detector_inference_speed_seconds) * 1000', 12, 1, unit="ms",
      desc="OpenVINO sobre CPU (pc1-node): normal ~15-25 ms. Alerta > 60 ms",
      thresholds=[{"color": "green", "value": None}, {"color": "orange", "value": 50}, {"color": "red", "value": 100}]),
 stat("Personas (24h)", f'sum(increase(frigate_camera_events_total{{camera="$camera",label="person"}}[24h]))', 16, 1, dec=0,
      thresholds=[{"color": "blue", "value": None}]),
 stat("Uptime Frigate", 'frigate_service_uptime_seconds', 20, 1, unit="s", dec=0,
      thresholds=[{"color": "blue", "value": None}]),
]
# --- Rendimiento
P.append(row("Rendimiento", 5))
P += [
 ts("FPS", [tgt(f'frigate_camera_fps{{{CAM}}}', "cámara", "A"),
            tgt(f'frigate_process_fps{{{CAM}}}', "procesados", "B"),
            tgt(f'frigate_detection_fps{{{CAM}}}', "detección", "C"),
            tgt(f'frigate_skipped_fps{{{CAM}}}', "saltados", "D")], 0, 6, minv=0),
 ts("Tiempo de inferencia", [tgt('frigate_detector_inference_speed_seconds * 1000', "{{name}}")], 12, 6, unit="ms", minv=0,
    desc="OpenVINO CPU en pc1-node. En la Pi 5 (TFLite) eran ~63 ms"),
]
# --- Recursos
P.append(row("Recursos (pc1-node)", 14))
P += [
 ts("CPU por proceso de Frigate", [tgt('frigate_cpu_usage_percent{type=~"Camera|detectors|go2rtc|recording|review_segment|embeddings"}', "{{name}} · {{process}}")],
    0, 15, unit="percent", stack=True, minv=0,
    desc="% de un core. ffmpeg = decodificar casona_sub por CPU (VAAPI de la HD 4600 disponible, no activado)"),
 ts("CPU y memoria del pod", [
      tgt('sum(rate(container_cpu_usage_seconds_total{namespace="cameras",container="frigate"}[5m]))', "CPU (cores)", "A")],
    12, 15, w=6, unit="none", minv=0, desc="Límite del pod: 4 cores"),
 ts("Memoria del pod", [
      tgt('sum(container_memory_working_set_bytes{namespace="cameras",container="frigate"})', "working set", "A")],
    18, 15, w=6, unit="bytes", minv=0, desc="Límite del pod: 4 GiB"),
 ts("Temperatura CPU pc1-node", [tgt('node_thermal_zone_temp{node="pc1-node",type="x86_pkg_temp"}', "pc1-node")], 0, 23, w=12, h=7,
    unit="celsius", desc="i7-4770: Tjmax 100 °C. Alertas en 85/95 °C"),
 ts("Red del pod (go2rtc → Frigate)", [
      tgt('sum(rate(container_network_receive_bytes_total{namespace="cameras",pod=~"frigate-.*"}[5m]))', "recibido", "A"),
      tgt('sum(rate(container_network_transmit_bytes_total{namespace="cameras",pod=~"frigate-.*"}[5m]))', "enviado", "B")],
    12, 23, w=12, h=7, unit="Bps", minv=0, desc="cAdvisor. Recibido ≈ casona (3MP, grabación) + casona_sub (720p, detección)"),
]
# --- Red de la cámara (ping blackbox desde rp1, job blackbox-icmp-camara, cada 10s)
PING = f'job="blackbox-icmp-camara",{CAM}'
P.append(row("Red de la cámara (WiFi)", 30))
corr = ts("Stream vs ping", [
      tgt(f'frigate_camera_fps{{{CAM}}}', "FPS cámara", "A"),
      tgt(f'100 * (1 - avg_over_time(probe_success{{{PING}}}[1m]))', "pérdida ping %", "B")],
    0, 31, w=24, h=8, minv=0,
    desc="Si el FPS cae a 0 junto con un pico de pérdida → la cámara pierde el WiFi. "
         "Si el FPS cae pero el ping sigue OK → falla el RTSP de la Tapo, no la red.")
corr["fieldConfig"]["overrides"] = [{
    "matcher": {"id": "byName", "options": "pérdida ping %"},
    "properties": [{"id": "custom.axisPlacement", "value": "right"}, {"id": "unit", "value": "percent"},
                   {"id": "min", "value": 0}, {"id": "max", "value": 100},
                   {"id": "custom.drawStyle", "value": "bars"}, {"id": "custom.fillOpacity", "value": 50},
                   {"id": "color", "value": {"mode": "fixed", "fixedColor": "red"}}]}]
P += [
 corr,
 ts("Pérdida de paquetes", [tgt(f'100 * (1 - avg_over_time(probe_success{{{PING}}}[1m]))', "pérdida (1m)")],
    0, 39, w=12, h=7, unit="percent", minv=0, maxv=100,
    desc="1 paquete ICMP cada 10s → ventana de 1m = 6 muestras"),
 ts("Latencia (RTT)", [tgt(f'probe_icmp_duration_seconds{{{PING},phase="rtt"}} * 1000', "RTT")],
    12, 39, w=12, h=7, unit="ms", minv=0,
    desc="Normal por WiFi 2.4 GHz: 2-10 ms. Picos altos sostenidos = interferencia o señal débil"),
]
# --- Eventos y storage
P.append(row("Eventos y almacenamiento", 46))
P += [
 ts("Eventos por hora", [tgt(f'sum by (label) (increase(frigate_camera_events_total{{camera="$camera"}}[1h]))', "{{label}}")],
    0, 47, unit="none", bars=True, stack=True, minv=0, h=8),
 gauge("Disco pc1-node (grabaciones)", '100 * frigate_storage_used_bytes{storage="/media/frigate/recordings"} / frigate_storage_total_bytes{storage="/media/frigate/recordings"}',
       12, 47, desc="LV /var/lib/rancher de pc1-node (HDD, 600 G)", warn=75, crit=90),
 gauge("/dev/shm", '100 * frigate_storage_used_bytes{storage="/dev/shm"} / frigate_storage_total_bytes{storage="/dev/shm"}',
       18, 47, w=3, desc="emptyDir 256Mi. Si se llena → 'Bus error'", warn=80, crit=95),
 gauge("/tmp/cache", '100 * frigate_storage_used_bytes{storage="/tmp/cache"} / frigate_storage_total_bytes{storage="/tmp/cache"}',
       21, 47, w=3, desc="emptyDir 1Gi en RAM (segmentos antes de validarse)", warn=80, crit=95),
]
dash = {
 "annotations": {"list": []}, "editable": True, "graphTooltip": 1, "links": [
   {"title": "Abrir Frigate", "type": "link", "url": "https://frigate.tailb0ca85.ts.net", "targetBlank": True}],
 "panels": P, "refresh": "30s", "schemaVersion": 39, "tags": ["frigate", "camaras"],
 "templating": {"list": [
   {"current": {}, "hide": 0, "includeAll": False, "label": "Datasource", "name": "datasource",
    "options": [], "query": "prometheus", "refresh": 1, "type": "datasource"},
   {"current": {"text": "casona", "value": "casona"}, "datasource": DS,
    "definition": "label_values(frigate_camera_fps, camera_name)", "hide": 0, "includeAll": False,
    "label": "Cámara", "multi": False, "name": "camera",
    "query": {"query": "label_values(frigate_camera_fps, camera_name)", "refId": "StandardVariableQuery"},
    "refresh": 2, "sort": 1, "type": "query"}]},
 "time": {"from": "now-6h", "to": "now"}, "timezone": "browser",
 "title": "Frigate", "uid": "frigate-homelab", "version": 1}

body = json.dumps(dash, indent=2, ensure_ascii=False)
out = sys.argv[1]
with open(out, "w") as f:
    f.write("# GENERADO por grafana/dashboards/gen-frigate-dashboard.py — no editar a mano\n")
    f.write("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: grafana-dashboard-frigate\n  namespace: monitoring\ndata:\n  frigate.json: |\n")
    for line in body.splitlines():
        f.write("    " + line + "\n")
print(f"{len(P)} paneles → {out}")
