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
* **Portable:** Un solo binario sin dependencias externas complejas.

## Instalación

### Desde el código fuente (requiere Go 1.20+)

```bash
git clone [https://github.com/tu-usuario/google-photos-backup.git](https://github.com/tu-usuario/google-photos-backup.git)
cd google-photos-backup
go build -o gpb main.go
```

## Configuración

La aplicación utiliza un navegador automatizado (Chrome/Chromium). La primera vez necesitarás iniciar sesión manualmente para guardar la sesión.

### Ejecutar el configurador

Ejecuta el siguiente comando en tu terminal:

```bash
./gpb configure
```

Sigue las instrucciones en pantalla. Esto generará un archivo de configuración en `~/.config/google-photos-backup/config.yaml`.

## Uso

(Próximamente)

## Créditos
Desarrollado por http://antonio.mg con ayuda de gemini