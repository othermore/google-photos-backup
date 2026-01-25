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

(Próximamente)

## Créditos
Desarrollado por http://ntonio.mg con ayuda de gemini