> [!IMPORTANT]
> Este proyecto está todavía en desarrollo. No lo uses, o hazlo bajo tu cuenta y riesgo. Espero tener un release inicial a finales de Marzo de 2026.


# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

Herramienta CLI para realizar copias de seguridad locales e incrementales de tu librería de Google Photos.

Diseñada para ejecutarse manualmente o vía Cron en servidores Linux (Debian, RedHat, etc.) y macOS.

## Características

* **Tres Modos de Operación:**
    *   **Sync**: Descarga directa interactiva desde Google Takeout.
    *   **Schedule + Drive**: Configura exportaciones recurrentes (cada 2 meses) a Drive y las descarga/sincroniza automáticamente usando `rclone`.
    *   **Import**: Procesa manualmente ZIPs de Takeout existentes.
*   **Pipeline de Almacenamiento Optimizado**: Descarga, Descompresión, Corrección, Deduplicación y Limpieza ocurren en flujo continuo para minimizar el uso de disco.
*   **Calidad Original**: Asegura la descarga de archivos originales con metadatos completos (fechas JSON corregidas).
*   **Deduplicación Inteligente**: Usa enlaces duros (hardlinks) para deduplicación entre snapshots (Cero Espacio para duplicados).
*   **Alertas por Email**: Te notifica si las copias de seguridad se vuelven obsoletas (vía sistema `msmtp`).
*   **Headless**: Configurable vía archivos, perfecto para servidores sin interfaz gráfica (GUI).

## Instalación

### Desde el código fuente (Requiere Go 1.20+)

```bash
git clone https://github.com/your-username/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
```

### Requisitos
*   **Google Chrome / Chromium**: Para la automatización del navegador (programación/solicitud).
*   **Rclone**: Requerido para el modo `drive` (descarga desde Google Drive).
*   **msmtp** (Opcional): Para alertas por correo electrónico.

## Configuración

Ejecuta el asistente de configuración:

```bash
./gpb configure
```

Esto configurará tu:
*   Directorio de Trabajo (espacio temporal)
*   Directorio de Backup (almacenamiento final)
*   Remoto de Rclone (para modo Drive)
*   Email para alertas

## Uso

### 1. Sincronización Interactiva (Sync)
Ideal para copias puntuales o ejecuciones iniciales. Inicia sesión en Google, solicita una descarga y crea una copia local.

```bash
./gpb sync
```

### 2. Backup Automatizado de Drive (Recomendado)
Este método es totalmente automatizado y robusto.

**Paso A: Programar Exportaciones Recurrentes**
Ejecuta esto **una vez** para configurar Google Takeout para exportar tus fotos a Drive cada 2 meses durante 1 año.

```bash
./gpb schedule
```

**Paso B: Sincronización Desatendida de Drive**
Ejecuta este comando vía **Cron** (ej. diariamente). Revisa tu Drive buscando nuevas exportaciones, las descarga, procesa y borra de Drive para ahorrar espacio en la nube.

> **Nota**: Agrupa inteligentemente los archivos por fecha y **espera a que la exportación esté completa** (detectada por la presencia del archivo `...-001.zip`) antes de descargar.

```bash
./gpb drive
```

**Ejemplo Cron:**
```bash
0 3 * * * /path/to/gpb drive >> /var/log/gpb.log 2>&1
```

### 3. Importación Manual
Si has descargado manualmente ZIPs de Takeout, puedes importarlos:

```bash
./gpb import /ruta/a/carpeta_con_zips
```

## Almacenamiento y Deduplicación

La herramienta organiza los archivos en una estructura `Backup/AAAA/MM`.
*   **Snapshots**: Cada ejecución puede actualizar la estructura existente o crear snapshots (configurable).
*   **Hardlinks**: Los archivos idénticos entre copias (o importados múltiples veces) se enlazan mediante hardlinks, sin usar espacio adicional.

## Solución de Problemas

*   **Login de Google**: Si `schedule` o `sync` se atascan en el login, ejecuta `gpb configure` y elige "Sí" para iniciar sesión interactivamente.
*   **Rclone**: Asegúrate de que `rclone lsd remote:` funciona antes de ejecutar `gpb drive`.
*   **Backups Obsoletos**: Si no has hecho copia en >90 días, `gpb drive` intentará primero **auto-renovar** la programación (headless, a menudo funciona sin Passkey). Si falla, enviará una alerta por email.

## Créditos

Desarrollado por http://antonio.mg con la ayuda de gemini