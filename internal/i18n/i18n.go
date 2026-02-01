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
