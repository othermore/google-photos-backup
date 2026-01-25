# Google Photos Backup (Linux/macOS)

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

Aplicación de línea de comandos (CLI) para realizar copias de seguridad locales e incrementales de tu biblioteca de Google Photos.

Diseñada para ser ejecutada manualmente o mediante Cron en servidores Linux (Debian, RedHat, etc.) y macOS.

## Características

* **Híbrido:** Utiliza la API oficial para listar archivos y un sistema de descarga directa para obtener la máxima calidad (saltando la compresión de la API).
* **Incremental:** Mantiene un índice local (`index.jsonl`) para descargar solo las fotos nuevas.
* **Headless:** Configurable mediante archivos, ideal para servidores sin interfaz gráfica.
* **Portable:** Un solo binario sin dependencias externas complejas.

## Instalación

### Desde el código fuente (requiere Go 1.20+)

Requisitos previos:
*   **Google Chrome** o **Chromium**: Debe estar instalado en el sistema para que funcione el descargador.
*   **Go 1.20+**: Para compilar.

```bash
git clone [https://github.com/tu-usuario/google-photos-backup.git](https://github.com/tu-usuario/google-photos-backup.git)
cd google-photos-backup
go build -o gpb main.go
```

## Configuración

Antes de usar la aplicación, necesitas obtener credenciales de Google.

### 1. Obtener Credenciales de Google

1.  Ve a la [Google Cloud Console](https://console.cloud.google.com/).
2.  Crea un **Nuevo Proyecto**.
3.  Habilita la **"Google Photos Library API"**.
4.  Configura la **Pantalla de consentimiento OAuth** (User Type: External). Añade tu email a "Usuarios de prueba".
5.  Crea **Credenciales** -> **ID de cliente de OAuth** (Tipo: Aplicación de escritorio).
6.  **Importante:** Añade `http://localhost:8085/callback` en "URI de redireccionamiento autorizados".
7.  Copia tu **ID de cliente** y tu **Secreto de cliente**.

### 2. Ejecutar el configurador

Ejecuta el siguiente comando en tu terminal:

```bash
./gpb configure
```

Sigue las instrucciones en pantalla. Esto generará un archivo de configuración en `~/.config/google-photos-backup/config.yaml`.

## Uso

### Comandos Disponibles

#### `configure`
Inicia el asistente interactivo para configurar credenciales y directorios.

```bash
./gpb configure
```

## Developer Info

Estructura del proyecto y descripción de ficheros. Mantener actualizada esta lista al añadir o modificar archivos.

*   `.gitignore`: Archivos ignorados por git.
*   `.project_context.md`: Contexto y reglas para el asistente de IA.
*   `cmd/`: Comandos de la aplicación (Cobra).
    *   `configure.go`: Lógica del comando `configure`.
    *   `root.go`: Punto de entrada de la CLI.
    *   `sync.go`: Lógica del comando `sync`.
    *   `utils.go`: Utilidades compartidas por los comandos.
*   `go.mod` / `go.sum`: Gestión de dependencias Go.
*   `internal/`: Código interno de la aplicación.
    *   `api/client.go`: Cliente HTTP para la Google Photos Library API.
    *   `auth/auth.go`: Flujo OAuth2 y gestión de tokens.
    *   `config/config.go`: Gestión de la configuración (viper).
    *   `downloader/browser.go`: Motor de automatización de navegador (go-rod).
    *   `i18n/i18n.go`: Sistema de internacionalización (EN/ES).
    *   `index/store.go`: Base de datos local para seguimiento de archivos.
    *   `utils/browser.go`: Utilidades generales del navegador (abrir URLs).
*   `main.go`: Entrypoint del binario.
*   `README.es.md`: Documentación en español.
*   `README.md`: Documentación en inglés.

## Créditos
Desarrollado por http://antonio.mg con ayuda de gemini