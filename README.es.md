# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

Herramienta CLI para realizar copias de seguridad locales e incrementales de tu librería de Google Photos.

Diseñada para ejecutarse manualmente o vía Cron en servidores Linux (Debian, RedHat, etc.) y macOS.

## Características

* **Takeout Automatizado:** Automatiza la solicitud y descarga de copias completas vía Google Takeout.
* **Calidad Original:** Asegura la descarga de archivos originales con metadatos completos.
* **Organización Inteligente:** Procesa los archivos descargados para corregir fechas EXIF (usando los JSONs de Google) y organiza fotos en álbumes.
* **Headless:** Configurable vía archivos, perfecto para servidores sin interfaz gráfica (GUI).
* **Descarga Robusta:**
    * Reanuda descargas interrumpidas automáticamente.
    * Maneja errores de "Quota Exceeded" marcando exportaciones como expiradas.
    * Valida tamaños de archivo antes de marcarlos como completos.
* **Experiencia de Usuario:**
    * Dashboard de progreso detallado con Velocidad y Tiempo Restante (ETA).
    * Opción de salida detallada (`verbose`) para depuración.
    * Internacionalizado (Inglés/Español).

## Instalación

### Desde el código fuente (Requiere Go 1.20+)

```bash
git clone https://github.com/your-username/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
```

## Configuración

La aplicación utiliza un navegador automatizado (Chrome/Chromium). Necesitarás iniciar sesión manualmente la primera vez para guardar la sesión.

### Setup

Ejecuta el asistente de configuración:

```bash
./gpb configure
```

Sigue las instrucciones. Esto autorizará a la herramienta a acceder a tus datos de Google Takeout.

### Archivo de Configuración

La configuración se almacena en `~/.config/google-photos-backup/config.yaml`.

## Uso

### 1. Sync (Comando Principal)

Comprueba el estado de tus exportaciones y maneja el flujo (Solicitar -> Esperar -> Descargar).

```bash
./gpb sync [flags]
```

**Flags:**

* `-v, --verbose`: Activa log detallado de depuración.
* `--force`: Ignora el chequeo de frecuencia (`backup_frequency`) y fuerza una nueva solicitud inmediatamente.

### 2. Process (Organización)

Una vez descargados los archivos, este comando extrae, corrige metadatos y organiza.

```bash
./gpb process [flags]
```

**Flujo de Trabajo:**
1.  **Extracción**: Descomprime nuevos archivos encontrados en `downloads/` a un subdirectorio `raw/`.
2.  **Corrección de Metadatos**: Usa los ficheros JSON de Google para corregir la "Fecha de Modificación" de tus imágenes/videos.
3.  **Deduplicación Global**: Escanea todos los ficheros, identifica duplicados (SHA256) y mantiene la mejor versión (priorizando nombres de álbum). Los duplicados se reemplazan con enlaces simbólicos relativos.

**Flags:**

*   `--force-metadata`: Re-ejecuta la corrección de fechas en exportaciones ya procesadas. Usa un escaneo "ligero" (rápido) sin recalcular hashes.
*   `--force-dedup`: Re-ejecuta la comprobación de duplicados. Fuerza un escaneo SHA256 completo para asegurar la integridad del índice.
*   `--force-extract`: Re-extrae los archivos comprimidos. Sobreescribe lo existente.
*   `--export <ID>`: Procesa SOLO el ID de exportación especificado.

## Ciclo de Vida de una Exportación

El sistema gestiona el ciclo de vida de cada Takeout mediante estados en `history.json`:

*   **`requested`**: Solicitud iniciada pero no confirmada por Google.
*   **`in_progress`**: Google está preparando los archivos.
*   **`ready`**: Archivos listos para descargar.
*   **`expired`**: La exportación caducó en los servidores de Google.
*   **`cancelled`**: Cancelada por el usuario o el sistema.
*   **`failed`**: Google falló al generar la exportación.

Desarrollado por http://antonio.mg con la ayuda de gemini