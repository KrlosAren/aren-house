# Systemd

## ¿Qué es?

Systemd es el sistema de inicialización y gestor de servicios de Linux. Reemplazó a SysVinit y es responsable de:
- Iniciar el sistema
- Gestionar servicios (daemons)
- Gestionar logs (journald)
- Gestionar red, tiempo, usuarios y más

## Conceptos Fundamentales

### Units

Todo en systemd se gestiona mediante "units". Tipos principales:

| Tipo | Extensión | Propósito |
|------|-----------|-----------|
| Service | `.service` | Procesos/daemons |
| Socket | `.socket` | Activación por socket |
| Target | `.target` | Agrupación de units |
| Timer | `.timer` | Programar ejecución (como cron) |
| Mount | `.mount` | Puntos de montaje |
| Path | `.path` | Activación por cambios en archivos |

### Estados de un servicio

| Estado | Significado |
|--------|-------------|
| `active (running)` | Ejecutándose |
| `active (exited)` | Terminó exitosamente |
| `inactive (dead)` | Detenido |
| `failed` | Falló |
| `activating` | Iniciando |

## Comandos Básicos

### Ver estado

```bash
# Estado de un servicio
sudo systemctl status dnsmasq

# Estado resumido
systemctl is-active dnsmasq

# ¿Está habilitado al boot?
systemctl is-enabled dnsmasq

# Listar todos los servicios
systemctl list-units --type=service

# Listar servicios fallidos
systemctl --failed
```

### Controlar servicios

```bash
# Iniciar
sudo systemctl start dnsmasq

# Detener
sudo systemctl stop dnsmasq

# Reiniciar
sudo systemctl restart dnsmasq

# Recargar configuración (sin reiniciar)
sudo systemctl reload dnsmasq

# Reiniciar si está activo
sudo systemctl try-restart dnsmasq
```

### Habilitar/Deshabilitar

```bash
# Iniciar automáticamente al boot
sudo systemctl enable dnsmasq

# No iniciar al boot
sudo systemctl disable dnsmasq

# Habilitar e iniciar ahora
sudo systemctl enable --now dnsmasq
```

### Recargar systemd

```bash
# Después de modificar unit files
sudo systemctl daemon-reload
```

## Ubicación de Unit Files

```
/lib/systemd/system/        # Units del sistema (paquetes)
/etc/systemd/system/        # Units personalizados (prioridad)
/run/systemd/system/        # Units temporales
```

## Ejemplo de Unit File

```ini
# /etc/systemd/system/mi-servicio.service
[Unit]
Description=Mi Servicio
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/mi-programa
Restart=always
User=admin

[Install]
WantedBy=multi-user.target
```

### Secciones

| Sección | Propósito |
|---------|-----------|
| `[Unit]` | Metadatos y dependencias |
| `[Service]` | Cómo ejecutar el servicio |
| `[Install]` | Cuándo habilitarlo |

## Journald (Logs)

Systemd incluye un sistema de logs llamado journald.

### Ver logs

```bash
# Logs de un servicio
sudo journalctl -u dnsmasq

# Seguir logs en tiempo real
sudo journalctl -u dnsmasq -f

# Logs desde el último boot
sudo journalctl -u dnsmasq -b

# Logs de las últimas 2 horas
sudo journalctl -u dnsmasq --since "2 hours ago"

# Logs de hoy
sudo journalctl -u dnsmasq --since today

# Solo errores
sudo journalctl -u dnsmasq -p err

# Todo el sistema
sudo journalctl -f
```

### Limpiar logs antiguos

```bash
# Mantener solo últimos 7 días
sudo journalctl --vacuum-time=7d

# Mantener solo 500MB
sudo journalctl --vacuum-size=500M
```

## Targets

Los targets agrupan units y definen estados del sistema.

| Target | Equivalente SysVinit | Estado |
|--------|---------------------|--------|
| `poweroff.target` | Runlevel 0 | Apagado |
| `rescue.target` | Runlevel 1 | Modo rescate |
| `multi-user.target` | Runlevel 3 | Multiusuario sin GUI |
| `graphical.target` | Runlevel 5 | Con GUI |
| `reboot.target` | Runlevel 6 | Reinicio |

```bash
# Ver target actual
systemctl get-default

# Cambiar target por defecto
sudo systemctl set-default multi-user.target

# Cambiar a un target ahora
sudo systemctl isolate rescue.target
```

## Dependencias

### Ver dependencias

```bash
# Qué necesita este servicio
systemctl list-dependencies dnsmasq

# Qué depende de este servicio
systemctl list-dependencies --reverse dnsmasq
```

### Definir dependencias en Unit

```ini
[Unit]
# Iniciar después de
After=network.target

# Requiere que esté activo
Requires=network.target

# Preferible pero no obligatorio
Wants=nss-lookup.target
```

## Servicios del Homelab

### dnsmasq

```bash
# Unit file
cat /lib/systemd/system/dnsmasq.service

# Estado
sudo systemctl status dnsmasq

# Logs
sudo journalctl -u dnsmasq -f
```

### nfs-kernel-server

```bash
sudo systemctl status nfs-kernel-server
sudo journalctl -u nfs-kernel-server
```

### WireGuard

WireGuard usa un template de unit:

```bash
# wg-quick@.service es un template
# wg-quick@wg0.service es la instancia para wg0

sudo systemctl status wg-quick@wg0
sudo journalctl -u wg-quick@wg0
```

## Timers (Alternativa a Cron)

```ini
# /etc/systemd/system/backup.timer
[Unit]
Description=Backup diario

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/backup.service
[Unit]
Description=Ejecutar backup

[Service]
Type=oneshot
ExecStart=/home/admin/backup.sh
```

```bash
# Habilitar timer
sudo systemctl enable --now backup.timer

# Ver timers activos
systemctl list-timers
```

## Troubleshooting

### Servicio no inicia

```bash
# Ver por qué falló
sudo systemctl status servicio
sudo journalctl -u servicio -n 50

# Ver logs del último intento
sudo journalctl -u servicio -b
```

### Servicio en loop de reinicio

```bash
# Ver intentos de reinicio
sudo journalctl -u servicio --since "10 min ago"

# Detener y revisar
sudo systemctl stop servicio
sudo systemctl status servicio
```

### Cambios en unit file no se aplican

```bash
# Siempre recargar después de editar
sudo systemctl daemon-reload
sudo systemctl restart servicio
```

## Referencias

- [systemd Documentation](https://www.freedesktop.org/wiki/Software/systemd/)
- [Arch Wiki - systemd](https://wiki.archlinux.org/title/systemd)
