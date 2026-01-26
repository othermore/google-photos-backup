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
