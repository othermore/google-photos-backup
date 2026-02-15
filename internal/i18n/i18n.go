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
	"prompt_download_mode": {
		"en": "Select download mode (%s/%s) [default: %s]",
		"es": "Selecciona el modo de descarga (%s/%s) [por defecto: %s]",
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
	"prompt_final_backup": {
		"en": "Final backup storage location (e.g. /nas/photos)",
		"es": "Ubicación final del backup (ej. /nas/fotos)",
	},
	"update_backup_start": {
		"en": "🔄 Starting backup update...",
		"es": "🔄 Iniciando actualización del backup...",
	},
	"update_backup_success": {
		"en": "\n✅ Backup successfully updated!\n   - Added: %d files (%s)\n   - Skipped: %d files\n   - Source processed: %s",
		"es": "\n✅ Backup actualizado correctamente!\n   - Añadidos: %d archivos (%s)\n   - Saltados: %d archivos\n   - Origen procesado: %s",
	},
	"update_backup_no_config": {
		"en": "❌ 'final_backup_path' is not configured. Run 'configure' first.",
		"es": "❌ 'final_backup_path' no está configurado. Ejecuta 'configure' primero.",
	},
	"process_start": {
		"en": "🚀 Starting Processing Phase",
		"es": "🚀 Iniciando Fase de Procesamiento",
	},
	"process_input": {
		"en": "📂 Input: %s",
		"es": "📂 Entrada: %s",
	},
	"process_output": {
		"en": "📂 Output: %s",
		"es": "📂 Salida: %s",
	},
	"process_albums": {
		"en": "📂 Albums: %s",
		"es": "📂 Álbumes: %s",
	},
	"process_fail": {
		"en": "❌ Processing failed: %v",
		"es": "❌ Procesamiento fallido: %v",
	},
	"process_success": {
		"en": "✅ Processing completed successfully.",
		"es": "✅ Procesamiento completado con éxito.",
	},
	"update_backup_source": {
		"en": "📂 Source Root: %s",
		"es": "📂 Origen (Root): %s",
	},
	"update_backup_source_missing": {
		"en": "❌ Source directory does not exist: %s",
		"es": "❌ El directorio de origen no existe: %s",
	},
	"update_backup_dest": {
		"en": "📂 Destination (Snapshot): %s",
		"es": "📂 Destino (Snapshot): %s",
	},
	"update_backup_dry_run": {
		"en": "⚠️ DRY RUN MODE: No files will be copied, linked, or deleted.",
		"es": "⚠️ MODO SIMULACIÓN: No se copiarán, enlazarán ni borrarán archivos.",
	},
	"update_backup_mkdir_fail": {
		"en": "❌ Failed to create snapshot directory: %v",
		"es": "❌ Error al crear directorio de snapshot: %v",
	},
	"update_backup_linking": {
		"en": "🔗 Linking unchanged files from previous backup: %s",
		"es": "🔗 Enlazando archivos sin cambios del backup anterior: %s",
	},
	"update_backup_history_loaded": {
		"en": "📜 Loaded history with %d exports.",
		"es": "📜 Historial cargado con %d exportaciones.",
	},
	"update_backup_history_fail": {
		"en": "⚠️ Could not load history.json (%v). Processing in directory order.",
		"es": "⚠️ No se pudo cargar history.json (%v). Procesando en orden de directorio.",
	},
	"update_backup_index_loaded": {
		"en": "✅ Loaded processing index from %s: %d completed exports, %d archives.",
		"es": "✅ Índice de procesamiento cargado desde %s: %d exportaciones completadas, %d archivos.",
	},
	"update_backup_index_missing": {
		"en": "⚠️ Could not find processing_index.json (checked %s). Validation will fail.",
		"es": "⚠️ No se pudo encontrar processing_index.json (comprobado %s). La validación fallará.",
	},
	"update_backup_implicit_complete": {
		"en": "⚠️ Export %s implicitly complete (all archives processed). Updating index...",
		"es": "⚠️ Exportación %s implícitamente completa (todos los archivos procesados). Actualizando índice...",
	},
	"update_backup_index_updated": {
		"en": "✅ processing_index.json updated.",
		"es": "✅ processing_index.json actualizado.",
	},
	"update_backup_skip_incomplete": {
		"en": "⚠️ Skipping incomplete export: %s (not fully processed)",
		"es": "⚠️ Saltando exportación incompleta: %s (no totalmente procesada)",
	},
	"update_backup_processing": {
		"en": "📦 Processing Archive: %s",
		"es": "📦 Procesando Archivo: %s",
	},
	"update_backup_fail_export": {
		"en": "❌ Failed to backup %s: %v",
		"es": "❌ Fallo al hacer backup de %s: %v",
	},
	"update_backup_delete_content": {
		"en": "🧹 Deleting media content for export: %s",
		"es": "🧹 Borrando contenido multimedia de exportación: %s",
	},
	"update_backup_delete_fail": {
		"en": "Failed to delete %s: %v",
		"es": "Fallo al borrar %s: %v",
	},
	"update_backup_dry_delete": {
		"en": "🧹 [Dry Run] Would delete: %s",
		"es": "🧹 [Simulación] Borraría: %s",
	},
	"update_backup_no_exports": {
		"en": "⚠️ No valid exports processed. Check 'process' status or source path.",
		"es": "⚠️ No se procesaron exportaciones válidas. Comprueba el estado de 'process' o la ruta de origen.",
	},
	"update_backup_log_updated": {
		"en": "📝 Backup log updated: %s",
		"es": "📝 Log de backup actualizado: %s",
	},
	"update_backup_summary_links": {
		"en": "   🔗 Hardlinks from previous: %d",
		"es": "   🔗 Hardlinks desde anterior: %d",
	},
	"update_backup_summary_internal": {
		"en": "   🔗 Internal hardlinks preserved: %d",
		"es": "   🔗 Hardlinks internos preservados: %d",
	},
	"update_backup_summary_exports": {
		"en": "   📦 Exports Processed: %d",
		"es": "   📦 Exportaciones Procesadas: %d",
	},
	"update_backup_copied": {
		"en": "➕ Copied: %s",
		"es": "➕ Copiado: %s",
	},
	"fix_hardlinks_start": {
		"en": "Starting Fix Hardlinks...",
		"es": "Iniciando Fix Hardlinks...",
	},
	"fix_hardlinks_scan": {
		"en": "📂 Scanning backups in: %s",
		"es": "📂 Escaneando backups en: %s",
	},
	"fix_hardlinks_dry": {
		"en": "⚠️ DRY RUN MODE",
		"es": "⚠️ MODO SIMULACIÓN",
	},
	"fix_hardlinks_not_enough": {
		"en": "Not enough snapshots to deduplicate.",
		"es": "No hay suficientes snapshots para deduplicar.",
	},
	"fix_hardlinks_analyze": {
		"en": "🔍 Analyzing snapshot: %s",
		"es": "🔍 Analizando snapshot: %s",
	},
	"fix_hardlinks_would_link": {
		"en": "Would link: %s -> %s",
		"es": "Enlazaría: %s -> %s",
	},
	"fix_hardlinks_complete": {
		"en": "✅ Fix Hardlinks Complete.",
		"es": "✅ Fix Hardlinks Completado.",
	},
	"fix_hardlinks_processed": {
		"en": "   Files Processed: %d",
		"es": "   Archivos Procesados: %d",
	},
	"fix_hardlinks_linked": {
		"en": "   Duplicates Linked: %d",
		"es": "   Duplicados Enlazados: %d",
	},
	"fix_hardlinks_saved": {
		"en": "   Space Saved: %s",
		"es": "   Espacio Ahorrado: %s",
	},
	"status_finalizing": {
		"en": "Finalizing",
		"es": "Finalizando",
	},
	"sync_history_error": {
		"en": "⚠️  Could not load history: %v",
		"es": "⚠️  No se pudo cargar el historial: %v",
	},
	"sync_ghost_removed": {
		"en": "🧹 Removed %d incomplete/ghost entries from history.",
		"es": "🧹 Eliminadas %d entradas incompletas/fantasma del historial.",
	},
	"sync_migrate_fail": {
		"en": "❌ Failed to migrate state: %v",
		"es": "❌ Fallo al migrar estado: %v",
	},
	"sync_found_completed": {
		"en": "✅ Found completed file: %s (Size: %s)",
		"es": "✅ Encontrado fichero completado: %s (Tamaño: %s)",
	},
	"sync_export_set": {
		"en": "📦 Export Set Detected: %d files, Total: %s",
		"es": "📦 Conjunto de exportación detectado: %d ficheros, Total: %s",
	},
	"sync_download_start": {
		"en": "⬇️  Starting: %s (%s)",
		"es": "⬇️  Iniciando: %s (%s)",
	},
	"sync_download_finish": {
		"en": "✅ Finished: %s (%s)",
		"es": "✅ Finalizado: %s (%s)",
	},
	"sync_quota_exceeded": {
		"en": "⛔ Download quota exceeded (Quota Exceeded).",
		"es": "⛔ Límite de descargas excedido (Quota Exceeded).",
	},
	"sync_quota_action": {
		"en": "⚠️  Marking export as EXPIRED and cleaning up partial data.",
		"es": "⚠️  Marcando exportación como EXPIRADA y limpiando datos parciales.",
	},
	"sync_cleanup_error": {
		"en": "❌ Error deleting download directory: %v",
		"es": "❌ Error eliminando directorio de descarga: %v",
	},
	"sync_cleanup_success": {
		"en": "🧹 Download directory deleted.",
		"es": "🧹 Directorio de descarga eliminado.",
	},
	"sync_new_export": {
		"en": "✅ New export created with ID: %s",
		"es": "✅ Nueva exportación creada con ID: %s",
	},
	"sync_pending_export": {
		"en": "⚠️  Export created but ID not yet visible. Saving as pending.",
		"es": "⚠️  Exportación creada pero ID aún no visible. Guardando como pendiente.",
	},
	"browser_waiting_content": {
		"en": "⏳ Waiting for page content...",
		"es": "⏳ Esperando contenido de la página...",
	},
	"browser_check_quota": {
		"en": "🔍 Checking for quota limit...",
		"es": "🔍 Comprobando límite de cuota...",
	},
	"browser_identify_pending": {
		"en": "🔍 Identifying pending files...",
		"es": "🔍 Identificando archivos pendientes...",
	},
	"browser_parse_url_fail": {
		"en": "⚠️  Failed to parse base URL %s: %v",
		"es": "⚠️  Fallo al analizar URL base %s: %v",
	},
	"browser_started_file": {
		"en": "\n     ... Started: %s",
		"es": "\n     ... Iniciado: %s",
	},
	"browser_unknown_start": {
		"en": "\n⚠️  Unknown download started: %s",
		"es": "\n⚠️  Descarga desconocida iniciada: %s",
	},
	"browser_js_fail": {
		"en": "❌ JS Execution failed for part %d: %v",
		"es": "❌ Ejecución JS falló para la parte %d: %v",
	},
	"browser_auth_prompt": {
		"en": "🔑 Auth prompt detected. Attempting to enter password...",
		"es": "🔑 Solicitud de autenticación detectada. Intentando introducir contraseña...",
	},
	"browser_no_pending": {
		"en": "✅ No pending files to download.",
		"es": "✅ No hay archivos pendientes para descargar.",
	},
	"browser_found_pending": {
		"en": "📋 Found %d pending files. Scraping URLs...",
		"es": "📋 Encontrados %d archivos pendientes. Extrayendo URLs...",
	},
	"browser_scraped_links": {
		"en": "📋 Scraped %d valid download links.",
		"es": "📋 Extraídos %d enlaces de descarga válidos.",
	},
	"browser_cleanup_incomplete": {
		"en": "🧹 Cleaning up incomplete download: %s",
		"es": "🧹 Limpiando descarga incompleta: %s",
	},
	"browser_all_tracked": {
		"en": "🏁 All downloads tracked as complete. Waiting 30s for file finalization...",
		"es": "🏁 Todas las descargas marcadas como completas. Esperando 30s para finalización de archivos...",
	},
	"browser_finished_failures": {
		"en": "🏁 Process finished (with some failures). Waiting 10s before closing...",
		"es": "🏁 Proceso finalizado (con algunos fallos). Esperando 10s antes de cerrar...",
	},
	"browser_firing_requests": {
		"en": "🚀 Firing download requests via Button Click (Robust JS)...",
		"es": "🚀 Lanzando peticiones de descarga vía Click (JS Robusto)...",
	},
	"browser_auth_challenge": {
		"en": "🔐 Auth/Passkey challenge detected! Waiting for user interaction...",
		"es": "🔐 ¡Reto de Auth/Passkey detectado! Esperando interacción del usuario...",
	},
	"browser_auth_instruction": {
		"en": "👉 Please complete the authentication in the browser window.",
		"es": "👉 Por favor completa la autenticación en la ventana del navegador.",
	},
	"browser_auth_timeout": {
		"en": "❌ Auth wait timed out.",
		"es": "❌ Tiempo de espera de autenticación agotado.",
	},
	"browser_auth_resolved": {
		"en": "✅ Auth resolved! Resuming...",
		"es": "✅ ¡Autenticación resuelta! Reanudando...",
	},
	"browser_quota_limit": {
		"en": "🔍 Checking for quota limit...",
		"es": "🔍 Comprobando límite de cuota...",
	},
	"browser_wait_redirect": {
		"en": "Waiting for redirect to Manage page...",
		"es": "Esperando redirección a la página de gestión...",
	},
	"browser_click_fail": {
		"en": "❌ Failed to click part %d: %v",
		"es": "❌ Fallo al hacer clic en la parte %d: %v",
	},
	"browser_detect_cancel": {
		"en": "⚠️  Detected 'Cancel export' button. Assuming export in progress.",
		"es": "⚠️  Detectado botón 'Cancelar exportación'. Asumiendo exportación en curso.",
	},
	"browser_detect_text": {
		"en": "⚠️  Detected in-progress text on page. Waiting.",
		"es": "⚠️  Detectado texto de 'en progreso' en la página. Esperando.",
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
