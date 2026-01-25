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
		"en": "To use this app, you need to create your own Google Cloud credentials",
		"es": "Para usar esta aplicación, necesitas crear tus propias credenciales",
	},
	"intro_2": {
		"en": "(it's free for personal use).",
		"es": "de Google Cloud (es gratuito para uso personal).",
	},
	"steps_title": {
		"en": "QUICK STEPS:",
		"es": "PASOS RÁPIDOS:",
	},
	"step_1": {
		"en": "1. Go to https://console.cloud.google.com/",
		"es": "1. Ve a https://console.cloud.google.com/",
	},
	"step_2": {
		"en": "2. Create a new project.",
		"es": "2. Crea un proyecto nuevo.",
	},
	"step_3": {
		"en": "3. Enable 'Google Photos Library API'.",
		"es": "3. Habilita la 'Google Photos Library API'.",
	},
	"step_4": {
		"en": "4. Configure OAuth Consent Screen (User Type: External).",
		"es": "4. Configura la Pantalla de consentimiento OAuth (User Type: External).",
	},
	"step_5": {
		"en": "5. Create OAuth 2.0 Credentials (Type: Desktop App).",
		"es": "5. Crea credenciales OAuth 2.0 (Tipo: Desktop App).",
	},
	"readme_hint": {
		"en": "Check README.md for a detailed step-by-step guide.",
		"es": "Consulta el README.md para una guía detallada paso a paso.",
	},
	"prompt_client_id": {
		"en": "Enter your Google Cloud Client ID: ",
		"es": "Introduce tu Google Cloud Client ID: ",
	},
	"prompt_client_secret": {
		"en": "Enter your Google Cloud Client Secret: ",
		"es": "Introduce tu Google Cloud Client Secret: ",
	},
	"prompt_backup_dir": {
		"en": "Backup directory (Press Enter to use '%s'): ",
		"es": "Directorio para guardar fotos (Enter para usar '%s'): ",
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