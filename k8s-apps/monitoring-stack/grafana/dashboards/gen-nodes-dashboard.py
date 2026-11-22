# Genera 11-grafana-dashboard-nodes.yml:
#   python3 k8s-apps/monitoring-stack/grafana/dashboards/gen-nodes-dashboard.py k8s-apps/monitoring-stack/grafana/11-grafana-dashboard-nodes.yml
#
# Dashboard "Nodos": hardware + software de rp1/rp2/rp3/pc1 desde node-exporter.
# Requiere el label "node" en el job node-exporter (02-prometheus-config.yml) y el
# collector systemd (homelab-ansible/playbooks/node-exporter.yml).
import json, sys

DS = {"type": "prometheus", "uid": "${datasource}"}
N = 'node=~"$node"'
_id = [0]


def nid():
    _id[0] += 1
    return _id[0]


def tgt(expr, legend="", ref="A", instant=False, fmt=None):
    t = {"datasource": DS, "expr": expr, "legendFormat": legend, "refId": ref}
    if instant:
        t["instant"] = True
    if fmt:
        t["format"] = fmt
    return t


def row(title, y):
    return {"type": "row", "title": title, "collapsed": False, "id": nid(), "panels": [],
            "gridPos": {"h": 1, "w": 24, "x": 0, "y": y}}


def steps(*pairs):
    return {"mode": "absolute", "steps": [{"color": c, "value": v} for c, v in pairs]}


def stat(title, expr, x, y, w=4, h=4, unit="none", dec=1, th=None, desc="", legend="{{node}}"):
    return {"type": "stat", "title": title, "description": desc, "id": nid(), "datasource": DS,
            "gridPos": {"h": h, "w": w, "x": x, "y": y},
            "targets": [tgt(expr, legend, instant=True)],
            "fieldConfig": {"defaults": {"unit": unit, "decimals": dec,
                                         "thresholds": th or steps(("green", None))}, "overrides": []},
            "options": {"colorMode": "background", "graphMode": "none", "justifyMode": "center",
                        "orientation": "horizontal", "textMode": "value_and_name",
                        "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False}}}


def ts(title, targets, x, y, w=12, h=8, unit="none", desc="", minv=None, maxv=None, th=None):
    d = {"unit": unit, "custom": {"drawStyle": "line", "fillOpacity": 10, "lineWidth": 2,
                                  "showPoints": "never", "spanNulls": True,
                                  "thresholdsStyle": {"mode": "dashed" if th else "off"}}}
    if minv is not None:
        d["min"] = minv
    if maxv is not None:
        d["max"] = maxv
    if th:
        d["thresholds"] = th
    return {"type": "timeseries", "title": title, "description": desc, "id": nid(), "datasource": DS,
            "gridPos": {"h": h, "w": w, "x": x, "y": y}, "targets": targets,
            "fieldConfig": {"defaults": d, "overrides": []},
            "options": {"legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
                        "tooltip": {"mode": "multi", "sort": "desc"}}}


def table(title, expr, x, y, w, h, keep, rename, desc=""):
    """Tabla instantánea con labels como columnas."""
    return {"type": "table", "title": title, "description": desc, "id": nid(), "datasource": DS,
            "gridPos": {"h": h, "w": w, "x": x, "y": y},
            "targets": [tgt(expr, instant=True, fmt="table")],
            "transformations": [
                {"id": "filterFieldsByName", "options": {"include": {"names": keep}}},
                {"id": "organize", "options": {"renameByName": rename,
                                               "indexByName": {k: i for i, k in enumerate(keep)}}}],
            "fieldConfig": {"defaults": {}, "overrides": []},
            "options": {"showHeader": True, "cellHeight": "sm",
                        "sortBy": [{"displayName": rename.get(keep[0], keep[0]), "desc": False}]}}


P = []

# ---------------------------------------------------------------- Resumen
P.append(row("Resumen", 0))
P += [
    stat("Uptime", f'time() - node_boot_time_seconds{{{N}}}', 0, 1, unit="s", dec=0,
         th=steps(("orange", None), ("green", 3600)), desc="Naranja = reiniciado hace menos de 1 hora"),
    stat("CPU", f'100 * (1 - avg by (node) (rate(node_cpu_seconds_total{{mode="idle",{N}}}[5m])))', 4, 1,
         unit="percent", th=steps(("green", None), ("orange", 70), ("red", 90))),
    stat("Memoria", f'100 * (1 - node_memory_MemAvailable_bytes{{{N}}} / node_memory_MemTotal_bytes{{{N}}})', 8, 1,
         unit="percent", th=steps(("green", None), ("orange", 75), ("red", 90))),
    stat("Disco /", f'100 * (1 - node_filesystem_avail_bytes{{mountpoint="/",fstype!~"tmpfs|overlay",{N}}} '
                    f'/ node_filesystem_size_bytes{{mountpoint="/",fstype!~"tmpfs|overlay",{N}}})', 12, 1,
         unit="percent", th=steps(("green", None), ("orange", 75), ("red", 90))),
    stat("Temperatura", f'node_thermal_zone_temp{{type=~"cpu-thermal|x86_pkg_temp",{N}}}', 16, 1, unit="celsius",
         th=steps(("green", None), ("orange", 70), ("red", 80)),
         desc="CPU (SoC en Pi, paquete en x86). Pi 5: throttling ~80-85 °C. i7-4770 (pc1-node): Tjmax 100 °C"),
    # max by (node): node_load1 trae instance/job/env y la división exige labels idénticos en ambos lados
    stat("Load (1m) / cores", f'max by (node) (node_load1{{{N}}}) / count by (node) (node_cpu_seconds_total{{mode="idle",{N}}})', 20, 1,
         dec=2, th=steps(("green", None), ("orange", 0.8), ("red", 1.5)),
         desc="> 1 = más trabajo del que los 4 cores pueden atender"),
]

# ---------------------------------------------------------------- Software
P.append(row("Software y servicios", 5))
P += [
    stat("Servicios caídos", f'count(node_systemd_unit_state{{state=~"failed|inactive",{N}}} == 1) or vector(0)',
         0, 6, w=4, h=5, dec=0, legend="", th=steps(("green", None), ("red", 1)),
         desc="Units vigilados (ver node_exporter_systemd_units en el playbook) que no están activos"),
    stat("Servicios en estado failed", f'count(node_systemd_unit_state{{state="failed",{N}}} == 1) or vector(0)',
         0, 11, w=4, h=4, dec=0, legend="", th=steps(("green", None), ("red", 1))),
    stat("NTP sincronizado", f'node_timex_sync_status{{{N}}}', 0, 15, w=4, h=5, dec=0,
         th=steps(("red", None), ("green", 1)), desc="1 = reloj sincronizado. Sin hora correcta fallan TLS y los certificados"),
    {   # Historial de estado de cada servicio
        "type": "state-timeline", "title": "Estado de servicios (systemd)", "id": nid(), "datasource": DS,
        "description": "Verde = activo, rojo = caído/failed. Lista de units: node_exporter_systemd_units en playbooks/node-exporter.yml. iscsid está enmascarado (playbooks/disable-iscsi.yml) y ya no se vigila.",
        "gridPos": {"h": 14, "w": 12, "x": 4, "y": 6},
        "targets": [tgt(f'max by (node, name) (node_systemd_unit_state{{state="active",{N}}})', "{{node}} · {{name}}")],
        "fieldConfig": {"defaults": {
            "color": {"mode": "thresholds"}, "thresholds": steps(("red", None), ("green", 1)),
            "mappings": [{"type": "value", "options": {
                "1": {"text": "activo", "color": "green", "index": 0},
                "0": {"text": "caído", "color": "red", "index": 1}}}],
            "custom": {"fillOpacity": 80, "lineWidth": 0}}, "overrides": []},
        "options": {"mergeValues": True, "showValue": "never", "rowHeight": 0.85, "alignValue": "left",
                    "legend": {"showLegend": False}, "tooltip": {"mode": "single"}}},
    table("Sistema operativo", f'node_os_info{{{N}}} * on (node) group_left(release) node_uname_info{{{N}}}',
          16, 6, 8, 5, keep=["node", "pretty_name", "release"],
          rename={"node": "Nodo", "pretty_name": "SO", "release": "Kernel"},
          desc="Un kernel distinto entre nodos suele indicar un apt upgrade a medias (falta reboot)"),
    stat("Reinicios (7 días)", f'changes(node_boot_time_seconds{{{N}}}[7d])', 16, 11, w=8, h=4, dec=0,
         th=steps(("green", None), ("orange", 2), ("red", 5)), desc="Reinicios inesperados = cortes de luz, kernel panic, OOM"),
    stat("Procesos bloqueados (I/O)", f'node_procs_blocked{{{N}}}', 16, 15, w=8, h=5, dec=0,
         th=steps(("green", None), ("orange", 2), ("red", 5)),
         desc="Procesos esperando disco. Fue el síntoma de los bloqueos iSCSI en rp3"),
]

# ---------------------------------------------------------------- CPU
P.append(row("CPU", 20))
P += [
    ts("Uso de CPU", [tgt(f'100 * (1 - avg by (node) (rate(node_cpu_seconds_total{{mode="idle",{N}}}[5m])))', "{{node}}")],
       0, 21, w=8, unit="percent", minv=0, maxv=100),
    ts("Frecuencia de CPU", [tgt(f'avg by (node) (node_cpu_scaling_frequency_hertz{{{N}}})', "{{node}}")],
       8, 21, w=8, unit="hertz", minv=0,
       desc="Pi 5: máx 2.4 GHz. Si baja con carga alta y temperatura alta = throttling térmico"),
    ts("Temperatura", [tgt(f'node_thermal_zone_temp{{type=~"cpu-thermal|x86_pkg_temp",{N}}}', "{{node}}")], 16, 21, w=8, unit="celsius",
       th=steps(("green", None), ("orange", 70), ("red", 80))),
    ts("Presión de CPU (PSI)", [tgt(f'rate(node_pressure_cpu_waiting_seconds_total{{{N}}}[5m]) * 100', "{{node}}")],
       0, 29, w=12, unit="percent", minv=0,
       desc="% del tiempo con procesos esperando CPU. Mejor indicador de saturación que el % de uso"),
    ts("Load average", [tgt(f'node_load1{{{N}}}', "{{node}} 1m", "A"), tgt(f'node_load15{{{N}}}', "{{node}} 15m", "B")],
       12, 29, w=12, minv=0, desc="4 cores por nodo: load sostenido > 4 = saturado"),
]

# ---------------------------------------------------------------- Memoria
P.append(row("Memoria", 37))
P += [
    ts("Memoria usada", [tgt(f'100 * (1 - node_memory_MemAvailable_bytes{{{N}}} / node_memory_MemTotal_bytes{{{N}}})', "{{node}}")],
       0, 38, w=8, unit="percent", minv=0, maxv=100),
    ts("Memoria disponible", [tgt(f'node_memory_MemAvailable_bytes{{{N}}}', "{{node}}")], 8, 38, w=8, unit="bytes", minv=0),
    ts("Presión de memoria (PSI)", [tgt(f'rate(node_pressure_memory_waiting_seconds_total{{{N}}}[5m]) * 100', "{{node}}")],
       16, 38, w=8, unit="percent", minv=0, desc="> 0 sostenido = el kernel está reclamando memoria; antesala del OOM killer"),
]

# ---------------------------------------------------------------- Disco
P.append(row("Disco", 46))
FS = f'fstype!~"tmpfs|overlay|squashfs|ramfs|nsfs",mountpoint!~"/run.*|/var/lib/(kubelet|docker).*|/var/lib/rancher/.+",{N}'
P += [
    ts("Uso de filesystems", [tgt(f'100 * (1 - node_filesystem_avail_bytes{{{FS}}} / node_filesystem_size_bytes{{{FS}}})',
                                  "{{node}} {{mountpoint}}")],
       0, 47, w=8, unit="percent", minv=0, maxv=100, th=steps(("green", None), ("orange", 75), ("red", 90))),
    ts("Lectura / escritura", [
        tgt(f'sum by (node) (rate(node_disk_read_bytes_total{{device=~"sd.*|mmcblk[0-9]+|nvme.*",{N}}}[5m]))', "{{node}} lectura", "A"),
        tgt(f'sum by (node) (rate(node_disk_written_bytes_total{{device=~"sd.*|mmcblk[0-9]+|nvme.*",{N}}}[5m]))', "{{node}} escritura", "B")],
       8, 47, w=8, unit="Bps", minv=0, desc="rp2: Frigate graba aquí; Prometheus también escribe en rp2"),
    ts("Presión de I/O (PSI)", [tgt(f'rate(node_pressure_io_waiting_seconds_total{{{N}}}[5m]) * 100', "{{node}}")],
       16, 47, w=8, unit="percent", minv=0, desc="% del tiempo esperando disco. Picos sostenidos = disco lento o saturado"),
]

# ---------------------------------------------------------------- Red
P.append(row("Red", 55))
NET = f'device=~"eth0|enx.*|tailscale0|wlan0",{N}'
P += [
    ts("Tráfico recibido", [tgt(f'rate(node_network_receive_bytes_total{{{NET}}}[5m])', "{{node}} {{device}}")],
       0, 56, w=8, unit="Bps", minv=0, desc="rp1: enx… = WAN (módem), eth0 = LAN del cluster, tailscale0 = VPN"),
    ts("Tráfico enviado", [tgt(f'rate(node_network_transmit_bytes_total{{{NET}}}[5m])', "{{node}} {{device}}")],
       8, 56, w=8, unit="Bps", minv=0),
    ts("Errores y descartes", [
        tgt(f'rate(node_network_receive_errs_total{{{NET}}}[5m]) + rate(node_network_transmit_errs_total{{{NET}}}[5m])', "{{node}} {{device}} errores", "A"),
        tgt(f'rate(node_network_receive_drop_total{{{NET}}}[5m]) + rate(node_network_transmit_drop_total{{{NET}}}[5m])', "{{node}} {{device}} descartes", "B")],
       16, 56, w=8, unit="pps", minv=0, desc="Debería ser ~0. Errores en eth0 = cable o puerto del switch"),
]

dash = {
    "annotations": {"list": []}, "editable": True, "graphTooltip": 1, "links": [],
    "panels": P, "refresh": "30s", "schemaVersion": 39, "tags": ["nodos", "node-exporter", "homelab"],
    "templating": {"list": [
        {"current": {}, "hide": 0, "includeAll": False, "label": "Datasource", "name": "datasource",
         "options": [], "query": "prometheus", "refresh": 1, "type": "datasource"},
        {"current": {"selected": True, "text": ["All"], "value": ["$__all"]}, "datasource": DS,
         "definition": "label_values(node_uname_info, node)", "hide": 0, "includeAll": True,
         "label": "Nodo", "multi": True, "name": "node",
         "query": {"query": "label_values(node_uname_info, node)", "refId": "StandardVariableQuery"},
         "refresh": 2, "sort": 1, "type": "query"}]},
    "time": {"from": "now-6h", "to": "now"}, "timezone": "browser",
    "title": "Nodos", "uid": "nodes-homelab", "version": 1}

body = json.dumps(dash, indent=2, ensure_ascii=False)
out = sys.argv[1]
with open(out, "w") as f:
    f.write("# GENERADO por grafana/dashboards/gen-nodes-dashboard.py — no editar a mano\n")
    f.write("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: grafana-dashboard-nodes\n  namespace: monitoring\ndata:\n  nodes.json: |\n")
    for line in body.splitlines():
        f.write("    " + line + "\n")
print(f"{len(P)} paneles → {out}")
