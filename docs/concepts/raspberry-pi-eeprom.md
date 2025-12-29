# Raspberry Pi EEPROM

## ¿Qué es?

La EEPROM (Electrically Erasable Programmable Read-Only Memory) del Raspberry Pi contiene el bootloader y la configuración de arranque. A diferencia del firmware en la microSD, la EEPROM persiste en el hardware.

## ¿Por qué es importante?

Para netboot, necesitamos configurar la EEPROM para que el Pi intente arrancar desde la red antes que desde la microSD.

## Configuración de Boot

### BOOT_ORDER

Define el orden en que el Pi intenta bootear. Es un número hexadecimal donde cada dígito representa un método:

| Dígito | Método |
|--------|--------|
| 0 | SD DETECT (interno) |
| 1 | SD/eMMC |
| 2 | Network |
| 3 | RPi Boot (USB device mode) |
| 4 | USB MSD |
| 5 | BCM-USB-MSD |
| 6 | NVME |
| 7 | HTTP |
| e | Stop |
| f | Restart |

### Ejemplo: 0xf2461

Leído de derecha a izquierda:
```
1 → SD/eMMC primero
6 → NVME segundo
4 → USB MSD tercero
2 → Network cuarto
f → Reiniciar si todo falla
```

### Configuración para Netboot (homelab)

```
BOOT_ORDER=0xf2461
```

Esto significa:
1. Intentar SD
2. Intentar NVME
3. Intentar USB
4. Intentar Network
5. Reiniciar si falla

Si quieres que intente red primero:
```
BOOT_ORDER=0xf21
```

## TFTP_PREFIX

Define cómo el Pi busca archivos en el servidor TFTP.

| Valor | Comportamiento |
|-------|----------------|
| 0 | Usa serial number |
| 1 | Usa "nombre de board" (obsoleto) |
| 2 | Usa serial number (8 chars) o MAC address |

En el homelab usamos `TFTP_PREFIX=2` que busca:
1. Por serial: `/440dc91d/start4.elf`
2. Por MAC: `/2c-cf-67-88-9e-f5/start4.elf`

## Comandos

### Ver configuración actual

```bash
sudo rpi-eeprom-config
```

Salida ejemplo:
```
[all]
BOOT_UART=1
WAKE_ON_GPIO=1
POWER_OFF_ON_HALT=0
BOOT_ORDER=0xf2461
TFTP_PREFIX=2
NET_INSTALL_AT_POWER_ON=0
```

### Modificar configuración

```bash
# Crear archivo de configuración
cat > /tmp/boot.conf << EOF
BOOT_ORDER=0xf2461
TFTP_PREFIX=2
NET_INSTALL_AT_POWER_ON=0
EOF

# Aplicar
sudo rpi-eeprom-config --apply /tmp/boot.conf
```

### Verificar versión de EEPROM

```bash
sudo rpi-eeprom-update
```

### Actualizar EEPROM

```bash
sudo rpi-eeprom-update -a
sudo reboot
```

## Opciones Adicionales

### BOOT_UART

Habilita salida de debug por UART durante el boot:
```
BOOT_UART=1
```

Útil para debuggear problemas de netboot.

### NET_INSTALL_AT_POWER_ON

Controla el instalador de red al encender:
```
NET_INSTALL_AT_POWER_ON=0  # Deshabilitado
```

### DHCP_TIMEOUT

Tiempo de espera para DHCP (en milisegundos):
```
DHCP_TIMEOUT=45000
```

### TFTP_IP

Forzar IP del servidor TFTP (normalmente se obtiene por DHCP):
```
TFTP_IP=10.0.0.1
```

## Flujo de Netboot

```
┌─────────────────────────────────────────────────────────────┐
│                    RASPBERRY PI BOOT                         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │    Lee EEPROM         │
              │    BOOT_ORDER=0xf2461 │
              └───────────┬───────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
    ┌─────────┐     ┌─────────┐     ┌─────────┐
    │   SD    │     │  NVME   │     │   USB   │
    │  ¿hay?  │     │  ¿hay?  │     │  ¿hay?  │
    └────┬────┘     └────┬────┘     └────┬────┘
         │ No            │ No            │ No
         └───────────────┴───────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │      Network (2)      │
              │   DHCP + TFTP boot    │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   DHCP: obtiene IP    │
              │   y servidor TFTP     │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   TFTP_PREFIX=2       │
              │   Busca: /SERIAL/ o   │
              │   /MAC-ADDRESS/       │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   Descarga:           │
              │   - start4.elf        │
              │   - config.txt        │
              │   - kernel            │
              │   - initrd            │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   Boot del kernel     │
              │   Monta NFS root      │
              └───────────────────────┘
```

## Estructura TFTP

Con `TFTP_PREFIX=2`, el Pi busca archivos en:

```
/srv/tftp/
├── 440dc91d/           # Serial del rp2
│   ├── config.txt
│   ├── start4.elf
│   └── current/
│       ├── cmdline.txt
│       ├── vmlinuz
│       └── initrd.img
└── 2c-cf-67-88-9e-f5 -> 440dc91d  # Symlink por MAC
```

## Troubleshooting

### Pi no intenta netboot

1. Verificar BOOT_ORDER:
   ```bash
   sudo rpi-eeprom-config | grep BOOT_ORDER
   ```

2. Verificar que Network (2) está en la secuencia

3. Verificar que no hay SD insertada (si SD está primero)

### Pi no encuentra servidor TFTP

1. Habilitar BOOT_UART para debug:
   ```bash
   echo "BOOT_UART=1" > /tmp/boot.conf
   sudo rpi-eeprom-config --apply /tmp/boot.conf
   ```

2. Conectar cable serial y ver salida

3. Verificar logs de dnsmasq:
   ```bash
   sudo tail -f /var/log/dnsmasq.log
   ```

### Pi encuentra TFTP pero no los archivos

1. Verificar estructura de directorios:
   ```bash
   ls -la /srv/tftp/
   ```

2. Verificar que el serial coincide:
   ```bash
   # En Pi (con microSD)
   grep Serial /proc/cpuinfo
   ```

3. Verificar symlinks por MAC:
   ```bash
   ls -la /srv/tftp/ | grep -- "->"
   ```

### Recuperar EEPROM corrupta

Si la EEPROM está corrupta, necesitas una microSD con:
1. Imagen de Raspberry Pi OS
2. El archivo `recovery.bin` (se incluye en las imágenes oficiales)

El Pi detecta `recovery.bin` y restaura la EEPROM.

## En el Homelab (Ansible)

El playbook `prepare-node.yml` configura la EEPROM:

```yaml
- name: Create EEPROM configuration file
  copy:
    content: |
      BOOT_ORDER=0xf2461
      TFTP_PREFIX=2
      NET_INSTALL_AT_POWER_ON=0
    dest: /tmp/boot.conf

- name: Apply EEPROM configuration
  command: rpi-eeprom-config --apply /tmp/boot.conf
```

## Referencias

- [Raspberry Pi Bootloader Configuration](https://www.raspberrypi.com/documentation/computers/raspberry-pi.html#raspberry-pi-bootloader-configuration)
- [Network Boot](https://www.raspberrypi.com/documentation/computers/remote-access.html#network-boot-your-raspberry-pi)
