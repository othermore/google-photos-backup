> [!IMPORTANT]
> **Pre-release v0.9.0**: La herramienta es estable y está muy cerca de la versión final 1.0.0, pero por favor, úsala bajo tu propia responsabilidad.
> 
> **👋 ¿Estás usando `gpb` o te ha parecido interesante?**
> ¡Nos encantaría saber de ti! Por favor, considera [dejarnos un mensaje en los Issues](https://github.com/othermore/google-photos-backup/issues) para decírnoslo o compartir tu opinión. ¡Tu apoyo es lo que nos motiva a seguir! Echa un vistazo a [CONTRIBUTING.md](CONTRIBUTING.md) para ver cómo puedes ayudar u opinar (en inglés).

[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![es](https://img.shields.io/badge/lang-es-yellow.svg)](README.es.md)

Herramienta CLI para mantener copias de seguridad locales e incrementales de tu librería de Google Photos y hacerlas accesibles desde **Immich**, con una mínima intervención del usuario.

Diseñada para ejecutarse manualmente o vía Cron en servidores Linux (Debian, RedHat, etc.) y macOS.
> **Nota sobre el uso "Desatendido"**: Debido a las políticas de seguridad de Google (Passkeys y re-autenticación), esta herramienta **no** es totalmente desatendida. Necesitarás interactuar manualmente con la ventana del navegador al menos una vez al año (en el mejor de los casos) cuando caduquen las exportaciones programadas o Google requiera reverificación.

## Características

    *   **Direct**: Configura exportaciones mediante "Enlace por correo", y la herramienta las descargará directamente comprobando Takeout periódicamente a través del navegador.
    *   **Drive**: Configura y automatiza exportaciones recurrentes a Google Drive usando `rclone`.
    *   **Import**: Procesa manualmente ZIPs de Takeout existentes.
    *   **Tool**: Herramientas técnicas para configuración, indexación e integraciones con Immich.
    *   **Pipeline de Almacenamiento Optimizado**: Descarga, Descompresión, Corrección, Deduplicación y Limpieza ocurren en flujo continuo para minimizar el uso de disco.
    *   **Calidad Original**: Asegura la descarga de archivos originales con metadatos completos (fechas JSON corregidas).
    *   **Deduplicación Inteligente**: Usa enlaces duros (hardlinks) para deduplicación entre snapshots (Cero Espacio para duplicados).
    *   **Integración con Immich**: Genera un repositorio de solo lectura `immich-master` para que tu backup pueda ser servido directamente por Immich sin duplicar datos.
    *   **Alertas por Email**: Te notifica si las copias de seguridad se vuelven obsoletas (ej. si Google deja de enviar exportaciones o requiere re-autenticación) vía sistema `msmtp`.
    *   **Headless**: Configurable vía archivos, perfecto para servidores sin interfaz gráfica (GUI).

## 🚀 Instalación y Compilación

Puedes instalar la herramienta descargando un binario precompilado o compilándola tú mismo desde el código fuente.

### Método 1: Descargar una Release (Recomendado)
Puedes descargar la última versión compilada para Linux (amd64, arm64) y macOS (Intel, Apple Silicon) directamente desde la **[página de Releases](https://github.com/othermore/google-photos-backup/releases)**.

1. Descarga el archivo `.tar.gz` correspondiente a tu sistema operativo.
2. Descomprímelo y coloca el archivo `gpb` en una ruta dentro de tu PATH (por ejemplo, `/usr/local/bin`).
3. Dale permisos de ejecución: `chmod +x /usr/local/bin/gpb`

### Método 2: Compilar desde el Código Fuente (Requiere Go 1.21+)

Alternativamente, puedes clonar el repositorio y compilar la herramienta:

```bash
git clone https://github.com/othermore/google-photos-backup.git
cd google-photos-backup
go build -o gpb main.go
chmod +x gpb
```

## Uso Básico

1. Crea un fichero `config.yaml` usando el ejemplo que verás más abajo.
2. Asegúrate de tener Chrome o Chromium instalado.
3. Si usas el modo `gpb drive`, autoriza `rclone` y haz una prueba con `rclone ls <tu_remoto>:Takeout`

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

## Herramienta de Backup Híbrido para Google Photos (gpb) v0.9.0

Una herramienta por línea de comandos escrita en Go que automatiza las copias de seguridad de Google Photos mediante Takeout. Descarga y extrae archivos de forma incremental en una estructura de carpetas por año/mes (`Backup/AAAA/MM`), mientras elimina automáticamente el **100% de los duplicados**, tanto dentro del lote actual como en todo el archivo histórico, utilizando "hardlinks" que ocupan un tamaño de cero bytes.

> **👋 ¿Estás usando `gpb` o te ha parecido interesante?**
> ¡Nos encantaría saber de ti! Por favor, considera [dejarnos un mensaje en los Issues](https://github.com/othermore/google-photos-backup/issues) para decírnoslo o compartir tu opinión. ¡Tu apoyo es lo que nos motiva! Echa un vistazo a [CONTRIBUTING.md](CONTRIBUTING.md) para ver cómo puedes ayudar (en inglés).

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

### 2. Backup Directo por Navegador (Método "Enlace por correo" de Takeout)
Ideal si no usas `rclone`. Aunque Takeout llama a esta opción "Enviar enlace de descarga por correo electrónico", esta herramienta no lee tus correos. En su lugar, comprueba periódicamente Google Takeout directamente a través del navegador para ver si se han generado nuevas exportaciones y las descarga de forma secuencial.

**Paso A: Programar Exportaciones**
* `gpb direct schedule`: Configura exportaciones recurrentes (cada 2 meses durante 1 año).
* `gpb direct schedule-once`: Configura una única exportación.

**Paso B: Descarga Directa Desatendida**
Ejecuta este comando vía **Cron** (ej. diariamente). Revisará de forma pasiva si hay nuevas exportaciones sin intervención del usuario y las procesará automáticamente.
```bash
./gpb direct download
```
> **Consejo**: Si pasan más de 2.5 meses (75 días) sin un backup exitoso (ej. la programación de exportaciones caducó), te alertará por email cada 7 días para que puedas volver a ejecutar el comando schedule manualmente (igual que el modo Drive).

### 3. Herramientas Técnicas
El comando `tool` agrupa todas las tareas de configuración y mantenimiento:
* `gpb tool configure`: Asistente interactivo de configuración.
* `gpb tool fix-metadata`: Aplica retroactivamente las fechas de modificación del sistema usando los JSON asociados.
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

## Configuración Avanzada (`config.yaml`)

La configuración se almacena en el archivo `config.yaml`. Por defecto, la herramienta lo buscará en las siguientes ubicaciones:
* **Linux**: `/etc/google-photos-backup/` o `~/.config/google-photos-backup/`
* **macOS**: `~/.config/google-photos-backup/` o en el directorio actual `./`

### Gestionar Varias Cuentas (El parámetro `--config-dir`)
Puedes gestionar backups de varias cuentas de Google distintas creando directorios de configuración separados para cada una. Esto asegura que los tokens del navegador estén aislados. Usa el parámetro global `--config-dir` con cualquier comando para especificar qué directorio utilizar:
```bash
# Ejemplo: Respaldando una segunda cuenta con su propia configuración
./gpb --config-dir ~/.config/gpb/cuenta2 tool configure
./gpb --config-dir ~/.config/gpb/cuenta2 drive download
```
Este parámetro es global y funciona con absolutamente todos los comandos (`drive`, `direct`, `tool`, `import`).

### 🖥️ Servidores Remotos (Workaround para Servidores sin Entorno Gráfico)
Si estás ejecutando `gpb` en un servidor "headless" sin entorno de escritorio (por ejemplo, a través de SSH), cualquier comando que requiera autenticación con Google (como `tool configure` o `drive request`) fallará o se quedará congelado al intentar abrir la ventana de Chromium para hacer login, ya que no tiene donde dibujar la interfaz gráfica (`Could not open the default X display`).

La solución es realizar el inicio de sesión inicial en tu PC personal y transferir la sesión (que es 100% portable):

1. **En tu ordenador local (Mac/Windows/Linux Desktop)**, ejecuta:
   ```bash
   ./gpb tool configure --config-dir ./config-servidor
   ```
2. El navegador se abrirá en tu PC. Inicia sesión en Google con normalidad y cierra la ventana al terminar.
3. Ahora tendrás creada una carpeta local llamada `config-servidor` con tu `config.yaml` y la subcarpeta `browser_data/` con la sesión autorizada.
4. **Transfiere esta carpeta completa de tu PC al servidor** mediante `scp`, `rsync` o FTP:
   ```bash
   scp -r ./config-servidor usuario@mi-servidor:~/config-servidor
   ```
5. ¡Ya está! Ahora puedes ejecutar todos los comandos remotos y desatendidos directamente en tu servidor apuntando a este directorio. El motor de Chromium arrancará de forma invisible (`headless`) y re-utilizará tu sesión sin pedir login:
   ```bash
   ./gpb drive download --config-dir ~/config-servidor
   ```

### Registro de Logs Global (El parámetro `--log`)
Puedes ordenar a la herramienta que escriba trazas de ejecución (en inglés) a un archivo de log estándar de Unix utilizando el parámetro global `--log`.
Esto es especialmente útil cuando se ejecuta la herramienta automáticamente vía Cron, ya que guarda un registro persistente de los archivos descargados, el progreso de extracción, los enlaces duplicados y cualquier error o resumen final.
```bash
# Ejemplo: Escribiendo logs de ejecución a un archivo
./gpb --log /var/log/gpb.log drive download
```
Este parámetro está soportado por los comandos `drive download`, `direct download` e `import`.

### Parámetros de Configuración
*   `working_path`: Directorio para archivos temporales, procesamiento y datos de sesión del navegador (`browser_data/`). **Pro tip**: Si tu destino final (`backup_path`) es un NAS remoto o un disco duro lento (HDD), configura tu `working_path` en un disco local rápido (SSD/NVMe). La herramienta usará esta unidad primero para descargar, descomprimir y hacer todo el esfuerzo de deduplicación pesado a máxima velocidad antes de transferir las fotos resultantes al almacenamiento lento, ahorrándote horas de red y cuello de botella.
*   `backup_path`: Destino final de las fotos organizadas (`Backup/AAAA/MM/...`).
*   `rclone_remote`: Nombre de tu remoto rclone (ej. `drive:`).
*   `email_alert_to`: Dirección de correo electrónico para recibir alertas de backups obsoletos (requiere `msmtp`).
*   `immich_master_enabled`: (`true`/`false`) Activa la integración del repositorio de solo lectura para Immich.
*   `immich_master_path`: Ruta donde se generará la carpeta `immich-master` (generalmente dentro de `backup_path`).
*   `fix_ambiguous_metadata`: (`yes`, `no`, `interactive`) Cómo manejar fotos con fechas JSON faltantes/ambiguas.
*   *Campos Heredados*: `client_id`, `client_secret` y `token_path` están obsoletos, ya que la autenticación ahora usa el navegador web directamente.

### Ejemplo de `config.yaml`
```yaml
working_path: "/var/lib/gpb/work"
backup_path: "/mnt/storage/photos"
rclone_remote: "gdrive:"
email_alert_to: "alertas@midominio.com"
immich_master_enabled: true
immich_master_path: "/mnt/storage/photos/immich-master"
fix_ambiguous_metadata: "yes"
```

### Detalles de Sesión y Autenticación
La herramienta utiliza un navegador Chrome/Chromium en segundo plazo (headless) para automatizar Google Takeout.
*   **¿Dónde se guarda?** Todas las cookies de sesión, tokens de confianza para Passkeys y logins se guardan en `[working_path]/browser_data`.
*   **Mantenlo a salvo**: No borres esta carpeta o necesitarás re-autenticarte manualmente (lo que podría requerir que conectes una Passkey física o un dispositivo 2FA de nuevo).

## Configuración de Herramientas del Sistema

### 1. Rclone (Para el Modo Drive)
Para usar `gpb drive`, necesitas `rclone` autorizado con tu cuenta de Google.
*   **macOS / Linux**: Instala con `sudo curl https://rclone.org/install.sh | sudo bash` o `brew install rclone`.
*   Ejecuta `rclone config`.
*   Crea un nuevo remoto (`n` de New remote). Llámalo exactamente igual a tu `rclone_remote` de `config.yaml` (por defecto es `drive`).
*   Selecciona `Google Drive` (`drive`).
*   Deja en blanco las credenciales de validación (Client credentials) o usa tus propias APIs si requieres límites altos.
*   Sigue el enlace en el navegador para dar acceso a Rclone a tu cuenta de Drive.

### 2. msmtp (Para Alertas de Correo)
La herramienta usa el binario `msmtp` nativo del sistema para enviarte correos si los backups tienen más de 2.5 meses (75 días).
*   **macOS**: `brew install msmtp`
*   **Linux (Debian/Ubuntu)**: `sudo apt install msmtp msmtp-mta`
*   Configura el fichero `~/.msmtprc` (o `/etc/msmtprc`) con tu servidor SMTP. Configuración de ejemplo para Gmail:
    ```ini
    defaults
    auth           on
    tls            on
    tls_trust_file /etc/ssl/certs/ca-certificates.crt
    
    account        default
    host           smtp.gmail.com
    port           587
    user           tu_correo@gmail.com
    password       tu_contraseña_de_aplicacion
    from           tu_correo@gmail.com
    ```
*   Dale permisos restrictivos: `chmod 600 ~/.msmtprc`

## Solución de Problemas

*   **Login de Google**: Si la automatización se atasca en el login, ejecuta `gpb tool configure` y elige "Sí" para iniciar sesión interactivamente.
*   **Rclone**: Asegúrate de que `rclone lsd remote:` funciona antes de ejecutar `gpb drive download`.
*   **Backups Obsoletos**: Si no has hecho un backup en >75 días, el comando de descarga enviará una alerta por correo cada 7 días recordándote que ejecutes manualmente el comando de programación (el cual requiere una Passkey).

## Créditos

Desarrollado por http://antonio.mg con la ayuda de gemini