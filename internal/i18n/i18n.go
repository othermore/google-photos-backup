package i18n

import (
	"os"
	"strings"
)

var CurrentLang = "en"

// Diccionario simple: Clave -> Mapa de idiomas
var messages = map[string]map[string]string{
	"header_title": {
		"en": "   GOOGLE PHOTOS BACKUP CONFIGURATION",
		"es": "   CONFIGURACIÓN DE GOOGLE PHOTOS BACKUP",
	},
	"intro_1": {
		"en": "This app automates Google Takeout backups.",
		"es": "Esta aplicación automatiza las copias de seguridad de Google Takeout.",
	},
	"intro_2": {
		"en": "You will need to log in via browser.",
		"es": "Necesitarás iniciar sesión a través del navegador.",
	},
	"steps_title": {
		"en": "QUICK STEPS:",
		"es": "PASOS RÁPIDOS:",
	},
	"step_1": {
		"en": "1. Run configure to set backup directory.",
		"es": "1. Ejecuta configure para establecer el directorio de backup.",
	},
	"prompt_backup_dir": {
		"en": "Backup directory",
		"es": "Directorio para guardar fotos",
	},
	"success_msg": {
		"en": "\n✅ Configuration saved to: %s",
		"es": "\n✅ Configuración guardada en: %s",
	},
	"error_mkdir": {
		"en": "Error creating config directory: %s",
		"es": "Error creando directorio de config: %s",
	},
	"error_save": {
		"en": "Error saving configuration: %s",
		"es": "Error guardando configuración: %s",
	},
	"prompt_fix_ambiguous": {
		"en": "Behavior for ambiguous metadata matches (yes/no/interactive) [default: %s]",
		"es": "Comportamiento para coincidencias de metadatos ambiguas (yes/no/interactive) [por defecto: %s]",
	},
	"login_ask": {
		"en": "\nDo you want to log in to Google now to validate access? (y/n)",
		"es": "\n¿Deseas iniciar sesión en Google ahora para validar el acceso? (s/n)",
	},
	"login_start": {
		"en": "Starting authentication flow...",
		"es": "Iniciando flujo de autenticación...",
	},
	"browser_open": {
		"en": "Opening test browser...",
		"es": "Abriendo navegador de prueba...",
	},
	"browser_nav_open": {
		"en": "ℹ️  Browser open. Please log in to your Google account.",
		"es": "ℹ️  Navegador abierto. Por favor, inicia sesión en tu cuenta de Google.",
	},
	"browser_nav_close": {
		"en": "ℹ️  When finished, simply close the browser window.",
		"es": "ℹ️  Cuando hayas terminado, simplemente cierra la ventana del navegador.",
	},
	"verifying_session": {
		"en": "🔍 Verifying session in background...",
		"es": "🔍 Verificando sesión en segundo plano...",
	},
	"navigating_takeout": {
		"en": "🚀 Navigating to Google Takeout...",
		"es": "🚀 Navegando a Google Takeout...",
	},
	"deselecting_products": {
		"en": "   - Deselecting all products...",
		"es": "   - Deseleccionando todos los productos...",
	},
	"selecting_photos": {
		"en": "   - Selecting Google Photos...",
		"es": "   - Seleccionando Google Photos...",
	},
	"next_step": {
		"en": "   - Proceeding to next step...",
		"es": "   - Avanzando al siguiente paso...",
	},
	"config_size": {
		"en": "   - Configuring size to 50GB...",
		"es": "   - Configurando tamaño a 50GB...",
	},
	"creating_export": {
		"en": "   - Creating export...",
		"es": "   - Creando la exportación...",
	},
	"waiting_confirmation": {
		"en": "   - Waiting for confirmation...",
		"es": "   - Esperando confirmación...",
	},
	"checking_status": {
		"en": "🔍 Checking export status in Takeout...",
		"es": "🔍 Comprobando estado de exportaciones en Takeout...",
	},
	"export_in_progress": {
		"en": "   - Detected Google Photos export in progress.",
		"es": "   - Detectada exportación en curso de Google Photos.",
	},
	"ignoring_other": {
		"en": "   - Ignoring export for another product.",
		"es": "   - Ignorando exportación en curso de otro producto.",
	},
	"cancelling_stale": {
		"en": "🛑 Cancelling stale export...",
		"es": "🛑 Cancelando exportación anterior (stale)...",
	},
	"cancel_sent": {
		"en": "   - Cancellation request sent.",
		"es": "   - Solicitud de cancelación enviada.",
	},
	"download_start": {
		"en": "⬇️  Starting download for export %s...",
		"es": "⬇️  Iniciando descarga de la exportación %s...",
	},
	"download_found": {
		"en": "   - Found %d files to download. Total size approx: %s",
		"es": "   - Encontrados %d archivos para descargar. Tamaño total aprox: %s",
	},
	"download_progress": {
		"en": "   - Downloading file %d/%d... (This may take a while)",
		"es": "   - Descargando archivo %d/%d... (Esto puede tardar)",
	},
	"download_skipped": {
		"en": "   - Part %d/%d already downloaded (%s). Skipping.",
		"es": "   - Parte %d/%d ya descargada (%s). Saltando.",
	},
	"download_success": {
		"en": "     ✅ Downloaded: %s",
		"es": "     ✅ Descargado: %s",
	},
	"sync_start": {
		"en": "Starting Google Takeout automation...",
		"es": "Iniciando automatización de Google Takeout...",
	},
	"sync_success": {
		"en": "\n✅ Request process finished. Google will email you when the export is ready.",
		"es": "\n✅ Proceso de solicitud finalizado. Google te enviará un email cuando la exportación esté lista.",
	},
	"sync_wait": {
		"en": "⏳ Export in progress. Google is preparing your files. Try again later.",
		"es": "⏳ Ya hay una exportación en curso. Google está preparando tus archivos. Vuelve a intentarlo más tarde.",
	},
	"config_missing": {
		"en": "Warning: config.yaml not found. Using defaults.",
		"es": "Aviso: No se encontró fichero config.yaml. Se usarán valores por defecto.",
	},
	"config_read_error": {
		"en": "Error reading config: %s",
		"es": "Error leyendo config: %s",
	},
	"config_decode_error": {
		"en": "Error decoding config: %s",
		"es": "Error decodificando config: %s",
	},
	"invalid_mode": {
		"en": "⚠️ Invalid mode, defaulting to %s",
		"es": "⚠️ Modo inválido, usando por defecto %s",
	},
	"validating_creds": {
		"en": "\nValidating saved credentials...",
		"es": "\nValidando credenciales guardadas...",
	},
	"session_valid": {
		"en": "\n✅ Session verified successfully. Future runs will use these cookies.",
		"es": "\n✅ Sesión guardada y verificada correctamente. Las próximas ejecuciones usarán estas cookies.",
	},
	"session_invalid": {
		"en": "\n⚠️  Could not verify session. Login might not have completed or 2FA is required.",
		"es": "\n⚠️  No se pudo verificar la sesión. Es posible que el login no se completara o Google pida 2FA de nuevo.",
	},
	"backup_dir_error": {
		"en": "❌ Error: Backup directory not configured. Please run './gpb configure' first.",
		"es": "❌ Error: El directorio de backup no está configurado. Por favor, ejecuta './gpb configure' primero.",
	},
	"backup_mkdir_error": {
		"en": "❌ Error creating backup directory: %v",
		"es": "❌ Error creando directorio de backup: %v",
	},
	"history_load_error": {
		"en": "⚠️  Could not load history: %v",
		"es": "⚠️  No se pudo cargar el historial: %v",
	},
	"status_check_error": {
		"en": "❌ Error checking status: %v",
		"es": "❌ Error comprobando estado: %v",
	},
	"merging_orphan": {
		"en": "   - Associating export %s with pending local request.",
		"es": "   - Asociando exportación %s a solicitud local pendiente.",
	},
	"importing_export": {
		"en": "   - Importing external export: %s (%s)",
		"es": "   - Importando exportación externa: %s (%s)",
	},
	"ignoring_expired": {
		"en": "⚠️  Ignoring expired export (previous Quota Exceeded): %s",
		"es": "⚠️  Ignorando exportación expirada (Quota Exceeded previo): %s",
	},
	"export_too_old": {
		"en": "⚠️  Export is too old (%v). It will be cancelled.",
		"es": "⚠️  La exportación lleva demasiado tiempo (%v). Se cancelará.",
	},
	"cancel_error": {
		"en": "❌ Error cancelling: %v",
		"es": "❌ Error cancelando: %v",
	},
	"ready_to_download": {
		"en": "🎉 Export ready for download!",
		"es": "🎉 ¡Exportación lista para descargar!",
	},
	"download_dir_error": {
		"en": "❌ Error creating download directory: %v",
		"es": "❌ Error creando directorio de descarga: %v",
	},
	"starting_manager": {
		"en": "🚀 Starting robust download manager...",
		"es": "🚀 Iniciando gestor de descargas robusto...",
	},
	"recovering_list": {
		"en": "📋 Recovering saved file list (%d files).",
		"es": "📋 Recuperando lista de ficheros guardada (%d ficheros).",
	},
	"list_error": {
		"en": "Error obtaining download list: %v",
		"es": "Error obteniendo lista de descarga: %v",
	},
	"state_save_error": {
		"en": "❌ Failed to save download state: %v",
		"es": "❌ Error guardando estado de descarga: %v",
	},
	"download_finished_error": {
		"en": "⚠️  Download finished with errors: %v",
		"es": "⚠️  La descarga finalizó con errores: %v",
	},
	"download_completed": {
		"en": "✅ Download completed. Files saved to: %s",
		"es": "✅ Descarga completada. Ficheros guardados en: %s",
	},
	"last_success": {
		"en": "✅ Last successful backup: %s",
		"es": "✅ La última copia exitosa fue el: %s",
	},
	"last_stats": {
		"en": "   Files: %d | Size: %s | New photos: %d",
		"es": "   Archivos: %d | Tamaño: %s | Nuevas fotos: %d",
	},
	"next_backup": {
		"en": "⏳ Too early for new backup (Freq: %s). Next: %s",
		"es": "⏳ No toca nueva copia todavía (Frecuencia: %s). Próxima: %s",
	},
	"use_force": {
		"en": "   Use --force to ignore this check.",
		"es": "   Usa --force para ignorar esta comprobación.",
	},
	"drive_mode_new": {
		"en": "⚠️  'driveDownload' mode configured. Creating new exports in this mode is not implemented yet.",
		"es": "⚠️  Modo 'driveDownload' configurado. La creación de nuevas exportaciones en este modo no está implementada aún.",
	},
	"takeout_req_error": {
		"en": "❌ Error during Takeout request: %v",
		"es": "❌ Error durante la solicitud de Takeout: %v",
	},
	"history_save_error": {
		"en": "❌ Error saving history: %v",
		"es": "❌ Error guardando historial: %v",
	},
	"history_updated": {
		"en": "📝 History updated at: %s",
		"es": "📝 Historial actualizado en: %s",
	},
	"browser_system": {
		"en": "ℹ️  Using system browser: %s",
		"es": "ℹ️  Usando navegador del sistema: %s",
	},
	"browser_download_fail": {
		"en": "⚠️  Failed to launch system browser. Trying to download Chromium...",
		"es": "⚠️  Falló al lanzar navegador del sistema. Intentando descargar Chromium...",
	},
	"progress_active": {
		"en": "Active",
		"es": "Activos",
	},
	"progress_done": {
		"en": "Done",
		"es": "Listos",
	},
	"progress_eta": {
		"en": "ETA",
		"es": "Tiempo",
	},
	"status_completed": {
		"en": "Completed",
		"es": "Completado",
	},
	"status_failed": {
		"en": "Failed",
		"es": "Fallido",
	},
	"status_downloading": {
		"en": "Downloading",
		"es": "Descargando",
	},
	"status_pending": {
		"en": "Pending",
		"es": "Pendiente",
	},
	"drive_mode_warning": {
		"en": "⚠️  'driveDownload' mode detected. Not supported yet.",
		"es": "⚠️  Modo 'driveDownload' detectado. Aún no está soportado.",
	},
	"quota_exceeded_limit": {
		"en": "⛔ Download quota exceeded (Quota Exceeded).",
		"es": "⛔ Límite de descargas excedido (Quota Exceeded).",
	},
	"quota_exceeded_action": {
		"en": "⚠️  Marking export as EXPIRED and cleaning up partial data.",
		"es": "⚠️  Marcando exportación como EXPIRADA y limpiando datos parciales.",
	},
	"discarding_bad_state": {
		"en": "⚠️  Discarding invalid legacy file list. Will re-scan.",
		"es": "⚠️  Descartando lista de archivos corrupta. Se re-escaneará.",
	},
	"migrating_state": {
		"en": "📦 Migrating download state to separate file...",
		"es": "📦 Migrando estado de descarga a fichero separado...",
	},
	"obtaining_list": {
		"en": "Obtaining download list...",
		"es": "Obteniendo lista de descarga...",
	},
	"list_saved": {
		"en": "✅ List saved: %d files.",
		"es": "✅ Lista guardada: %d ficheros.",
	},
}

// Init detecta el idioma del sistema
func Init() {
	// En Linux/Mac, la variable LANG suele ser "es_ES.UTF-8", "en_US.UTF-8", etc.
	langEnv := os.Getenv("LANG")
	if strings.HasPrefix(langEnv, "es") {
		CurrentLang = "es"
	} else {
		CurrentLang = "en"
	}
}

// T traduce una clave al idioma actual
func T(key string) string {
	if translations, ok := messages[key]; ok {
		if val, ok := translations[CurrentLang]; ok {
			return val
		}
		// Fallback a inglés si falta la traducción específica
		return translations["en"]
	}
	return key // Devuelve la clave si no existe
}
