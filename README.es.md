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

**Opciones Disponibles:**

| Clave | Valor por Defecto | Descripción |
| :--- | :--- | :--- |
| `working_path` | `./work` | Directorio de trabajo para descargas y archivos temporales. |
| `backup_frequency` | `168h` | Frecuencia para solicitar nuevas copias (ej. `24h`, `168h` = 7 días). |
| `download_mode` | `directDownload` | Modo de operación. Actualmente solo se soporta `directDownload`. |
| `fix_ambiguous_metadata` | `interactive` | Comportamiento para coincidencias ambiguas (`yes`, `no`, `interactive`). |
| `backup_path` | (vacío) | Ruta al destino final de la copia de seguridad (snapshots). |
| `user_data_dir` | (auto) | Ruta al perfil de usuario de Chrome (no cambiar salvo necesario). |

## Ciclo de Vida de una Exportación

El sistema gestiona el ciclo de vida de cada Takeout mediante estados en `history.json`:

*   **`requested`**: Solicitud iniciada pero no confirmada por Google.
*   **`in_progress`**: Google está preparando los archivos.
*   **`ready`**: Archivos listos para descargar.
*   **`expired`**: La exportación caducó en los servidores de Google.
*   **`cancelled`**: Cancelada por el usuario o el sistema.
*   **`failed`**: Google falló al generar la exportación.

### Integridad de Datos

Para asegurar la seguridad y evitar conflictos:
1.  **`history.json`**: Fuente de la verdad de las exportaciones. Modificado por `sync`, solo lectura para `process`.
2.  **`state.json`**: Rastrea el progreso de cada ZIP.
3.  **`processing_index.json`**: Rastrea qué exportaciones y archivos han sido procesados.

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
    *   **Niveles de Coincidencia**:
        *   **Nivel 1 (Exacto)**: Coincide `archivo.json` o `archivo.supplemental-metadata.json`.
        *   **Nivel 2 (Limpio)**: Coincide eliminando extensión (ej. `IMG_123.jpg` -> `IMG_123.json`).
        *   **Nivel 3 (Difuso/Seguro)**: Coincide nombres truncados si la longitud común es **>40 caracteres** (evita que `IMG.json` coincida con `IMG_1234.jpg`).
    *   **Coincidencias Ambiguas**: Las coincidencias parciales menores de 40 caracteres usan el comportamiento definido por `--fix-ambiguous-metadata`.
3.  **Deduplicación Global**: Escanea todos los ficheros, identifica duplicados (SHA256) y mantiene la mejor versión. Los duplicados se reemplazan con enlaces simbólicos relativos para ahorrar espacio.

**Flags:**

*   `--fix-ambiguous-metadata`: Comportamiento para coincidencias ambiguas (`yes`=aplicar, `no`=saltar, `interactive`=preguntar). Por defecto: `interactive`.
*   `--force-metadata`: Re-ejecuta la corrección de fechas en exportaciones ya procesadas.
*   `--force-dedup`: Re-ejecuta la comprobación de duplicados.
*   `--force-extract`: Re-extrae los archivos comprimidos.
*   `--export <ID>`: Procesa SOLO el ID de exportación especificado.

### 3. Actualizar Backup (Sincronización Final)

Tras procesar, este comando sincroniza los archivos organizados con tu ubicación de almacenamiento final (ej. NAS, disco externo).

```bash
./gpb update-backup [flags]
```

**Características:**
*   **Snapshots:** Crea una carpeta con fecha/hora para cada ejecución (`AAAA-MM-DD-HHMMSS`). **Soporta sufijos** (ej. `2024-05-20-173000-MiEtiqueta`).
*   **Deduplicación Inteligente:** Comprueba si los archivos ya existen en la *copia anterior*. Si el contenido coincide (Hash), crea un **enlace duro (hardlink)** en lugar de copiar. ¡Ahorra mucho espacio!
*   **Registro (Log):** Guarda detalles exactos de cada operación en `backup_log.jsonl` en el directorio final.
*   **Limpieza:** Si finaliza con éxito, borra los archivos procesados del `working_path` para liberar espacio.

**Flags:**
*   `--dry-run`: Simula la actualización sin copiar ni borrar nada.
*   `--source <dir>`: Especifica manualmente el directorio origen (por defecto `working_path/downloads`).

### 4. Fix Hardlinks (Deduplicación)

Escanea tus snapshots antiguos o modificados manualmente para maximizar el ahorro de espacio enlazando archivos idénticos entre copias.

```bash
./gpb fix-hardlinks [flags]
```

**Flags:**
*   `--dry-run`: Simula la deduplicación sin modificar archivos.
*   `--path <dir>`: Ruta a la raíz del backup (por defecto `backup_path`).

### 5. Directorio Maestro para Immich (Opcional)

Puedes mantener una estructura de directorios aplanada y deduplicada, optimizada para **Immich** (o cualquier biblioteca externa). Esta carpeta organiza todas tus fotos por Año/Mes usando enlaces duros (hardlinks), por lo que **no ocupa espacio adicional**.

**Configuración (en `config.yaml`):**
```yaml
immich_master_enabled: true
immich_master_path: "immich-master" # relativo a backup_path
```

**Características:**
*   **Auto-Actualización:** Al ejecutar `update-backup`, las nuevas fotos se enlazan automáticamente al directorio maestro.
*   **Estructura:** `immich-master/AAAA/MM/foto.jpg`.
*   **Reconstrucción (Rebuild):** Puedes generar este directorio desde tus snapshots existentes en cualquier momento:
    ```bash
    ./gpb rebuild-immich-master
    ```

## Solución de Problemas

*   **Problemas de Login:** Si la herramienta se queda atascada verificando la sesión, prueba a ejecutar `./gpb configure` de nuevo.
*   **"Quota Exceeded":** Si ocurre este error, la herramienta marcará la exportación como expirada y solicitará una nueva.
*   **Modo Verbose:** Ejecuta con `./gpb sync -v`.

## Créditos

Desarrollado por http://antonio.mg con la ayuda de gemini