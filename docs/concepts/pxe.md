# PXE (Preboot eXecution Environment)

## Qué es

Estándar que permite a un computador arrancar desde la red, sin necesidad de disco duro, USB o microSD. Pronunciado "pixie".

## Para qué sirve

- **Despliegue masivo**: Instalar SO en muchas máquinas sin tocar cada una
- **Clientes ligeros**: Computadores sin almacenamiento local (thin clients)
- **Raspberry Pi sin SD**: Arrancar RPi directamente desde la red
- **Recuperación**: Boot de rescate cuando el disco falla

## Cómo funciona

PXE combina varios protocolos:

```
[Dispositivo sin SO]                    [Servidor]
        |                                    |
        |  1. DHCP Discover                  |
        |  "Necesito IP y dónde bootear"     |
        |----------------------------------->|
        |                                    |
        |  2. DHCP Offer                     |
        |  "IP: 10.0.0.105                   |
        |   TFTP: 10.0.0.1                   |
        |   Archivo: bootcode.bin"           |
        |<-----------------------------------|
        |                                    |
        |  3. TFTP Request                   |
        |  "Dame bootcode.bin"               |
        |----------------------------------->|
        |                                    |
        |  4. Archivos de boot               |
        |  [kernel, initrd, config]          |
        |<-----------------------------------|
        |                                    |
        |  5. Sistema arranca                |
        |  (puede montar NFS para rootfs)    |
```

### Componentes necesarios

| Componente | Función | En homelab |
|------------|---------|------------|
| DHCP | Asigna IP + indica servidor boot | dnsmasq |
| TFTP | Sirve archivos de arranque | dnsmasq |
| NFS (opcional) | Sistema de archivos raíz | Futuro |

## En el homelab

### Raspberry Pi Network Boot

Las RPi 3B+ y 4 soportan netboot nativo:

```bash
# Habilitar netboot en RPi (una sola vez, desde SD)
sudo raspi-config
# Advanced Options → Boot Order → Network Boot

# O via comando
sudo rpi-eeprom-config --edit
# Agregar: BOOT_ORDER=0xf21  (SD → USB → Network)
```

### Configuración dnsmasq para PXE

```bash
# /etc/dnsmasq.conf

# DHCP con opciones PXE
dhcp-range=10.0.0.100,10.0.0.200,24h
dhcp-boot=bootcode.bin

# Para RPi específicamente (por serial number)
# dhcp-host=dc:a6:32:xx:xx:xx,rp2,10.0.0.2,set:rpi
# dhcp-boot=tag:rpi,bootcode.bin

# TFTP
enable-tftp
tftp-root=/srv/tftp
```

### Estructura de archivos RPi

```
/srv/tftp/
├── bootcode.bin          # Stage 1: GPU bootloader
├── start4.elf            # Stage 2: GPU firmware (RPi4)
├── fixup4.dat            # Memoria GPU
├── bcm2711-rpi-4-b.dtb   # Device tree
├── kernel8.img           # Kernel ARM64
├── config.txt            # Configuración boot
├── cmdline.txt           # Parámetros kernel
└── overlays/             # Device tree overlays
```

Ver: [ADR-003 dnsmasq](../decisions/003-dnsmasq-dhcp-dns-tftp.md)

## Flujo detallado para RPi

1. **Power on**: RPi busca boot en SD → USB → Red
2. **DHCP**: Obtiene IP y servidor TFTP
3. **bootcode.bin**: GPU descarga y ejecuta
4. **start4.elf**: Inicializa GPU completamente
5. **config.txt**: Lee configuración
6. **kernel8.img**: Carga kernel en RAM
7. **cmdline.txt**: Pasa parámetros al kernel
8. **Boot**: Kernel arranca, puede montar rootfs via NFS

## Comandos útiles

```bash
# Ver si RPi tiene netboot habilitado
vcgencmd bootloader_config | grep BOOT_ORDER

# Monitorear peticiones DHCP/TFTP
sudo journalctl -u dnsmasq -f

# Capturar tráfico PXE (debug)
sudo tcpdump -i eth0 port 67 or port 69

# Verificar archivos de boot
ls -la /srv/tftp/
```

## Troubleshooting

| Problema | Causa probable | Solución |
|----------|---------------|----------|
| No obtiene IP | DHCP no responde | Verificar dnsmasq, firewall |
| Timeout TFTP | Archivos no encontrados | Revisar tftp-root, permisos |
| Kernel panic | cmdline.txt incorrecto | Verificar root=, nfsroot= |
| Se queda en arcoiris | GPU firmware corrupto | Reemplazar start4.elf |
