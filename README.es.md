> [!IMPORTANT]
> Este proyecto está todavía en desarrollo. No lo uses, o hazlo bajo tu cuenta y riesgo. Espero tener un release inicial a finales de Marzo de 2026.


# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

Herramienta CLI para mantener copias de seguridad locales e incrementales de tu librería de Google Photos y hacerlas accesibles desde **Immich**, con una mínima intervención del usuario.

Diseñada para ejecutarse manualmente o vía Cron en servidores Linux (Debian, RedHat, etc.) y macOS.
> **Nota sobre el uso "Desatendido"**: Debido a las políticas de seguridad de Google (Passkeys y re-autenticación), esta herramienta **no** es totalmente desatendida. Necesitarás interactuar manualmente con la ventana del navegador al menos una vez al año (en el mejor de los casos) cuando caduquen las exportaciones programadas o Google requiera reverificación.

## Características

* **Cuatro Modos de Operación:**
    *   **Direct**: Configura y descarga archivos directamente a través de enlaces de correo electrónico.
    *   **Drive**: Configura y automatiza exportaciones recurrentes a Google Drive usando `rclone`.
    *   **Import**: Procesa manualmente ZIPs de Takeout existentes.
    *   **Tool**: Herramientas técnicas para configuración, indexación e integraciones con Immich.
*   **Pipeline de Almacenamiento Optimizado**: Descarga, Descompresión, Corrección, Deduplicación y Limpieza ocurren en flujo continuo para minimizar el uso de disco.
*   **Calidad Original**: Asegura la descarga de archivos originales con metadatos completos (fechas JSON corregidas).
*   **Deduplicación Inteligente**: Usa enlaces duros (hardlinks) para deduplicación entre snapshots (Cero Espacio para duplicados).
*   **Integración con Immich**: Genera un repositorio de solo lectura `immich-master` para que tu backup pueda ser servido directamente por Immich sin duplicar datos.
*   **Alertas por Email**: Te notifica si las copias de seguridad se vuelven obsoletas (ej. si Google deja de enviar exportaciones o requiere re-autenticación) vía sistema `msmtp`.
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
./gpb tool configure
```

Esto configurará tu:
*   Directorio de Trabajo (espacio temporal)
*   Directorio de Backup (almacenamiento final)
*   Remoto de Rclone (para modo Drive)
*   Email para alertas

## Uso

### 1. Backup Automatizado de Drive (Recomendado)
Este método es totalmente automatizado y robusto. Utiliza Google Drive para almacenar archivos temporales antes de descargarlos usando `rclone`, procesarlos secuencialmente y borrarlos de la nube para ahorrar espacio.

**Paso A: Programar Exportaciones**
Ejecuta esto para configurar Google Takeout para exportar tus fotos a Drive.
* `gpb drive schedule`: Configura exportaciones recurrentes (cada 2 meses durante 1 año).
* `gpb drive schedule-once`: Configura una única exportación.

**Paso B: Descarga Desatendida de Drive**
Ejecuta este comando vía **Cron** (ej. diariamente). Revisa tu Drive buscando nuevas exportaciones y las procesa sin intervención.
```bash
./gpb drive download
```
**Ejemplo Cron:**
```bash
0 3 * * * /path/to/gpb drive download >> /var/log/gpb.log 2>&1
```

### 2. Backup Directo por Email
Ideal si no usas `rclone`. Configura exportaciones mediante correo electrónico y las descarga secuencialmente.

**Paso A: Programar Exportaciones**
* `gpb direct schedule`: Configura exportaciones recurrentes por Email.
* `gpb direct schedule-once`: Configura una única exportación por Email.

**Paso B: Descarga Directa**
Espera hasta recibir el correo electrónico de Google, y luego ejecuta:
```bash
./gpb direct download
```
> **Consejo**: Puedes ejecutar `gpb direct download` diariamente vía Cron. Revisará de forma pasiva si hay nuevas exportaciones y las procesará. Si pasan más de 2 meses (60 días) sin un backup exitoso, te alertará por email (igual que el modo Drive).

### 3. Herramientas Técnicas
El comando `tool` agrupa todas las tareas de configuración y mantenimiento:
* `gpb tool configure`: Asistente interactivo de configuración.
* `gpb tool rebuild-index`: Reconstruye los índices locales.
* `gpb tool fix-hardlinks`: Valida y repara los enlaces duros entre volúmenes.
* `gpb tool rebuild-immich-master`: Sincroniza un snapshot con un repositorio de solo lectura `immich-master`.

### 4. Importación Manual
Si has descargado manualmente ZIPs de Takeout, puedes importarlos directamente:
```bash
./gpb import /ruta/a/carpeta_con_zips
```

## Almacenamiento y Deduplicación

La herramienta organiza los archivos en una estructura `Backup/AAAA/MM`.
*   **Snapshots**: Cada ejecución puede actualizar la estructura existente o crear snapshots (configurable).
*   **Hardlinks**: Los archivos idénticos entre copias (o importados múltiples veces) se enlazan mediante hardlinks, sin usar espacio adicional.

## Solución de Problemas

*   **Login de Google**: Si la automatización se atasca en el login, ejecuta `gpb tool configure` y elige "Sí" para iniciar sesión interactivamente.
*   **Rclone**: Asegúrate de que `rclone lsd remote:` funciona antes de ejecutar `gpb drive download`.
*   **Backups Obsoletos**: Si no has hecho copia en >90 días, `gpb drive download` intentará primero **auto-renovar** la programación (headless, a menudo funciona sin Passkey). Si falla, enviará una alerta por email.

## Créditos

Desarrollado por http://antonio.mg con la ayuda de gemini