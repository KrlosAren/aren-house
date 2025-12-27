# Conceptos de Netboot en Raspberry Pi

Guía conceptual sobre cómo funciona el boot por red (PXE/NFS) en Raspberry Pi.

## Arquitectura General
```
rp1-master (Gateway)
├── dnsmasq
│   ├── DHCP: Asigna IPs fijas por MAC
│   ├── DNS: Resuelve .homelab.local
│   └── TFTP: Sirve archivos de boot (kernel, initrd, dtb)
│
├── NFS Server
│   ├── /srv/nfs/rp2/  → Root filesystem de rp2
│   └── /srv/nfs/rp3/  → Root filesystem de rp3
│
└── /srv/tftp/
    ├── 440dc91d/      → Archivos boot de rp2 (por serial)
    ├── 02671e08/      → Archivos boot de rp3 (por serial)
    ├── 2c-cf-67-88-9e-f5 → Symlink a 440dc91d (por MAC de rp2)
    └── 2c-cf-67-a9-b9-13 → Symlink a 02671e08 (por MAC de rp3)
```

## Flujo de Boot por Red
```
1. Raspberry Pi enciende sin microSD
         │
         ▼
2. EEPROM busca boot por red (BOOT_ORDER=0xf2461)
         │
         ▼
3. Envía solicitud DHCP
         │
         ▼
4. dnsmasq responde con:
   - IP (10.0.0.3)
   - TFTP server (10.0.0.1)
         │
         ▼
5. Pi busca archivos en TFTP:
   - Primero por MAC: /srv/tftp/2c-cf-67-a9-b9-13/
   - O por serial: /srv/tftp/02671e08/ (si TFTP_PREFIX=2)
         │
         ▼
6. Descarga: config.txt → vmlinuz → initrd.img → cmdline.txt
         │
         ▼
7. cmdline.txt dice: root=/dev/nfs nfsroot=10.0.0.1:/srv/nfs/rp3
         │
         ▼
8. Kernel monta NFS como root filesystem
         │
         ▼
9. Sistema arranca normalmente
```

## Estructura de Carpetas TFTP
```
/srv/tftp/
├── 02671e08/                  ← Directorio por SERIAL (últimos 8 chars)
│   ├── config.txt             ← Configuración de boot
│   ├── cmdline.txt            ← Parámetros del kernel (o en current/)
│   └── current/               ← Subdirectorio con archivos de boot
│       ├── vmlinuz            ← Kernel
│       ├── initrd.img         ← Initial ramdisk
│       ├── bcm2712-rpi-5-b.dtb ← Device tree
│       └── overlays/          ← Overlays del hardware
│
└── 2c-cf-67-a9-b9-13 → symlink a 02671e08  ← Por MAC (fallback)
```

### ¿Por qué el symlink por MAC?

El EEPROM puede buscar archivos de dos formas:
- `TFTP_PREFIX=2` → Busca por serial (02671e08/)
- `TFTP_PREFIX=0` → Busca por MAC (2c-cf-67-a9-b9-13/)

Aunque configuramos `TFTP_PREFIX=2`, a veces la Pi busca primero por MAC. El symlink garantiza que encuentre los archivos en cualquier caso.

## Archivos Críticos

### config.txt
```ini
[all]
os_prefix=current/    ← Le dice dónde buscar vmlinuz, initrd, dtb
kernel=vmlinuz
cmdline=cmdline.txt
initramfs initrd.img followkernel
```

**os_prefix explicado:**
- Si `os_prefix=` está vacío, busca en la raíz del directorio
- Si `os_prefix=current/`, busca en el subdirectorio current/

Ubuntu organiza los archivos en `current/`, por eso necesitamos `os_prefix=current/`.

### cmdline.txt
```
console=serial0,115200 console=tty1 root=/dev/nfs nfsroot=10.0.0.1:/srv/nfs/rp3,vers=3,tcp rw ip=dhcp rootwait
```

| Parámetro | Significado |
|-----------|-------------|
| `root=/dev/nfs` | El root filesystem es NFS (no disco local) |
| `nfsroot=10.0.0.1:/srv/nfs/rp3` | Servidor y ruta NFS |
| `vers=3,tcp` | Versión NFS 3 sobre TCP |
| `ip=dhcp` | Obtener IP por DHCP antes de montar |
| `rootwait` | Esperar a que el root esté disponible |

Este archivo es **crítico**. El original de la microSD apunta a disco local y debe cambiarse para NFS.

## Estructura NFS (Root Filesystem)
```
/srv/nfs/rp3/
├── bin/
├── etc/
│   ├── fstab         ← Modificado para NFS
│   ├── hostname      ← Cambiado a "rp3-node"
│   ├── shadow        ← Copiado de la microSD (credenciales)
│   └── sudoers.d/
│       └── admin     ← Sudo sin contraseña
├── home/
│   └── admin/        ← Usuario con UID 1000
├── proc/             ← Directorio vacío (se monta en runtime)
├── sys/              ← Directorio vacío (se monta en runtime)
├── dev/              ← Dispositivos básicos creados manualmente
│   ├── null
│   ├── zero
│   ├── random
│   ├── urandom
│   ├── console
│   └── tty
└── ...
```

### /etc/fstab para NFS boot
```
proc            /proc   proc    defaults          0 0
sysfs           /sys    sysfs   defaults          0 0
tmpfs           /tmp    tmpfs   defaults          0 0
tmpfs           /run    tmpfs   defaults          0 0
```

Solo puntos de montaje virtuales. No hay referencias a discos porque no hay disco local.

## Configuración EEPROM
```bash
BOOT_ORDER=0xf2461    # Orden de boot
TFTP_PREFIX=2         # Buscar por serial (no por MAC)
NET_INSTALL_AT_POWER_ON=0  # No instalar automáticamente
```

### BOOT_ORDER explicado
```
0xf2461 = f 2 4 6 1
          │ │ │ │ └── 1 = SD Card
          │ │ │ └──── 6 = NVMe
          │ │ └────── 4 = USB
          │ └──────── 2 = Network
          └────────── f = Retry loop
```

Si no hay microSD, intenta NVMe, luego USB, luego red. Si todo falla, reinicia y reintenta.

## Cambios Necesarios y Por Qué

| Cambio | Por qué fue necesario |
|--------|----------------------|
| **Renombrar usuario a admin** | UID 1000 consistente en todas las máquinas para NFS |
| **Crear tempadmin** | No puedes renombrar un usuario mientras está logueado |
| **Copiar /etc/shadow** | Las credenciales de SSH están aquí, sin esto no puedes hacer login |
| **Crear /etc/sudoers.d/admin** | Ansible necesita sudo sin contraseña |
| **Configurar eth0 en netplan** | Ubuntu solo tenía WiFi configurado, eth0 estaba apagado |
| **Agregar regla NAT en iptables** | Sin esto, los nodos no tienen internet |
| **Crear symlink por MAC** | Fallback si TFTP_PREFIX no funciona como esperado |
| **Modificar os_prefix en config.txt** | Apuntar al subdirectorio donde están los archivos |
| **Modificar cmdline.txt** | Cambiar de boot local a boot NFS |
| **Crear directorios proc/sys/dev** | El kernel necesita estos puntos de montaje |
| **Crear dispositivos en /dev** | El sistema necesita null, zero, console para arrancar |
| **Modificar /etc/fstab** | Eliminar referencias a disco local, solo mountpoints virtuales |
| **Cambiar /etc/hostname** | Para que el nodo tenga el nombre correcto |

## Problemas Comunes y Soluciones

| Problema | Causa | Solución |
|----------|-------|----------|
| "Permission denied (publickey)" | /etc/shadow no tiene credenciales | Copiar shadow desde microSD |
| "sudo: I can't do that" | Falta archivo sudoers | Crear /etc/sudoers.d/admin |
| No encuentra archivos TFTP | os_prefix incorrecto | Ajustar config.txt |
| eth0 sin IP | No configurado en netplan | Agregar eth0 con dhcp4: true |
| Sin internet | Falta regla NAT | iptables MASQUERADE |
| Busca por MAC no por serial | TFTP_PREFIX no aplicado | Crear symlink por MAC |

## Relación entre Componentes
```
┌─────────────────────────────────────────────────────────────────┐
│                        RASPBERRY PI                             │
│                                                                  │
│  EEPROM                                                         │
│  ├── BOOT_ORDER=0xf2461  → Define orden de búsqueda de boot     │
│  └── TFTP_PREFIX=2       → Busca por serial en TFTP             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 1. DHCP Request
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        DNSMASQ                                   │
│                                                                  │
│  DHCP                                                           │
│  ├── Asigna IP por MAC                                          │
│  └── Informa TFTP server                                        │
│                                                                  │
│  TFTP                                                           │
│  └── Sirve /srv/tftp/{serial}/ o /srv/tftp/{mac}/               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 2. Descarga kernel + initrd
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ARCHIVOS DE BOOT                             │
│                                                                  │
│  /srv/tftp/{serial}/                                            │
│  ├── config.txt      → os_prefix=current/                       │
│  └── current/                                                   │
│      ├── vmlinuz     → Kernel Linux                             │
│      ├── initrd.img  → Initial ramdisk                          │
│      ├── cmdline.txt → root=/dev/nfs nfsroot=10.0.0.1:/srv/nfs/ │
│      └── *.dtb       → Device tree                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 3. Kernel lee cmdline.txt
                              │    y monta NFS como root
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        NFS SERVER                                │
│                                                                  │
│  /srv/nfs/rp3/                                                  │
│  ├── bin/, lib/, usr/  → Sistema operativo                      │
│  ├── etc/                                                       │
│  │   ├── fstab         → Solo montajes virtuales                │
│  │   ├── shadow        → Credenciales (copiado de microSD)      │
│  │   └── sudoers.d/    → Permisos sudo                          │
│  ├── home/admin/       → Home del usuario (UID 1000)            │
│  └── dev/, proc/, sys/ → Montados en runtime                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Resumen

El netboot elimina la dependencia de microSD en los nodos worker. El gateway (rp1-master) centraliza:

1. **DHCP/TFTP** → Asigna IP y sirve archivos de boot
2. **NFS** → Proporciona el sistema de archivos completo

Cada nodo necesita:
- Directorio TFTP con su serial (y symlink por MAC)
- Directorio NFS con su root filesystem
- EEPROM configurado para boot por red
- cmdline.txt apuntando a su directorio NFS específico

La ventaja principal: si un nodo falla, solo hay que conectar otro Raspberry Pi con el EEPROM configurado y booteará automáticamente con el mismo sistema.
