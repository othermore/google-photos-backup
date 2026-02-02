# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

Aplicación de línea de comandos (CLI) para realizar copias de seguridad locales e incrementales de tu biblioteca de Google Photos.

Diseñada para ser ejecutada manualmente o mediante Cron en servidores Linux (Debian, RedHat, etc.) y macOS.

## Características

* **Takeout Automatizado:** Automatiza la solicitud y descarga de copias de seguridad completas desde Google Takeout.
* **Calidad Original:** Garantiza la descarga de los archivos originales con todos sus metadatos.
* **Organización Inteligente:** Procesa los archivos descargados para corregir fechas EXIF (usando los JSON de Google) y organiza las fotos en álbumes.
* **Headless:** Configurable mediante archivos, ideal para servidores sin interfaz gráfica.
* **Descarga Robusta:**
    * Reanuda descargas interrumpidas automáticamente.
    * Gestiona errores de "Cuota Excedida" marcando exportaciones como expiradas.
    * Valida el tamaño de los ficheros antes de marcarlos como completados.
* **Experiencia de Usuario:**
    * Panel de progreso detallado con Velocidad y Tiempo Restante (ETA).
    * Opción de salida detallada (verbose) para depuración.
    * Internacionalizado (Inglés/Español).

## Instalación

### Desde el código fuente (requiere Go 1.20+)

```bash
git clone https://github.com/tu-usuario/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
```

## Configuración

La aplicación utiliza un navegador automatizado (Chrome/Chromium). La primera vez necesitarás iniciar sesión manualmente para guardar la sesión.

### Configuración Inicial

Ejecuta el asistente de configuración:

```bash
./gpb configure
```

Sigue las instrucciones en pantalla para autorizar a la herramienta a acceder a tus datos de Google Takeout.

### Archivo de Configuración

La configuración se guarda en `~/.config/google-photos-backup/config.yaml`.

**Opciones Disponibles:**

| Clave | Por Defecto | Descripción |
| :--- | :--- | :--- |
| `backup_path` | `./backup` | Directorio principal donde se guardarán las copias. |
| `backup_frequency` | `168h` | Frecuencia para solicitar nuevas copias (ej: `24h`, `168h` = 7 días). |
| `download_mode` | `directDownload` | Modo de operación. Actualmente solo se soporta `directDownload`. |
| `user_data_dir` | (auto) | Ruta al perfil de usuario de Chrome (no cambiar salvo necesidad). |

## Uso

### 1. Sincronizar (Comando Principal)

Comprueba el estado de tus exportaciones y gestiona el flujo (Solicitar -> Esperar -> Descargar).

```bash
./gpb sync [flags]
```

**Flags (Opciones):**

* `-v, --verbose`: Activa el registro detallado (muestra clics del navegador, URLs de navegación, etc.).
* `--force`: Ignora la comprobación de `backup_frequency` y fuerza una nueva solicitud de exportación inmediatamente.

### Cómo Funciona

1.  **Comprobar Estado:** La herramienta inicia sesión en Google Takeout para buscar exportaciones activas.
    *   **En Progreso:** Si se está creando una, espera. Si lleva más de 48h, la cancela.
    *   **Lista:** Si hay una lista (`Download` disponible), comienza a descargar.
    *   **Ninguna:** Si no hay exportación activa y ha pasado el tiempo de frecuencia, solicita una nueva (50GB divididos).
2.  **Descarga:**
    *   Los archivos se descargan inicialmente en la carpeta de Descargas del sistema.
    *   Se mueven automáticamente a `<backup_path>/downloads/<ID_EXPORTACION>/`.
    *   Un archivo `.download_state.json` rastrea el progreso, permitiendo reanudar si se interrumpe.
3.  **Finalización:**
    *   Una vez descargados todos los archivos, se actualiza el índice `history.json`.

## Solución de Problemas

*   **Problemas de Login:** Si la herramienta se atasca verificando la sesión, prueba a ejecutar `./gpb configure` de nuevo y asegúrate de completar el login en la ventana del navegador.
*   **"Quota Exceeded" (Cuota Excedida):** Google limita el número de veces que puedes descargar un archivo Takeout (usualmente 5-10 veces). Si ocurre este error, la herramienta marcará la exportación como expirada y solicitará una nueva en la siguiente ejecución.
*   **Modo Verbose:** Ejecuta con `gpb sync -v` para ver exactamente qué está haciendo la automatización del navegador.

## Créditos
Desarrollado por http://antonio.mg con ayuda de gemini