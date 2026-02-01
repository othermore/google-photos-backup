package browser

import (
	"fmt"
	"google-photos-backup/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google-photos-backup/internal/registry"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	URLGoogleHome      = "https://www.google.com"
	URLGoogleLogin     = "https://accounts.google.com"
	URLGooglePhotos    = "https://photos.google.com"
	URLTakeoutSettings = "https://takeout.google.com/settings/takeout?hl=en"
	URLTakeoutManage   = "https://takeout.google.com/manage?hl=en"
	URLTakeoutArchive  = "https://takeout.google.com/manage/archive/%s?hl=en"
)

// Manager gestiona la instancia del navegador y la sesión
type Manager struct {
	Browser *rod.Browser
	DataDir string // Directorio para guardar cookies y sesión
}

// New crea una nueva instancia del gestor del navegador
func New(userDataDir string, headless bool) *Manager {
	// Intentamos buscar el navegador del sistema primero (Chrome instalado)
	path, _ := launcher.LookPath()

	// Configuramos el lanzador
	l := launcher.New().
		UserDataDir(userDataDir). // Persistencia de sesión
		Headless(headless).
		Set("lang", "en-US"). // Force English locale
		Devtools(false).
		Set("disable-blink-features", "AutomationControlled"). // Ocultar que es un bot
		Set("exclude-switches", "enable-automation").          // Evita la barra "Chrome is being controlled..."
		Set("use-automation-extension", "false")               // Desactiva extensión de automatización

	if path != "" {
		fmt.Printf("ℹ️  Usando navegador del sistema: %s\n", path)
		l = l.Bin(path)
	}

	// Si no es headless (modo login), aseguramos que la ventana sea visible
	if !headless {
		l = l.Set("start-maximized")
	}

	// Lanzamos el navegador
	url, err := l.Launch()
	if err != nil {
		// Si falla, intentamos buscar el ejecutable del sistema o descargarlo
		fmt.Printf("⚠️  Falló al lanzar navegador del sistema. Intentando descargar Chromium...\n")
		// Recreamos el launcher básico para descargar
		l = launcher.New().
			UserDataDir(userDataDir).
			Headless(headless).
			Set("disable-blink-features", "AutomationControlled").
			Set("exclude-switches", "enable-automation").
			Set("use-automation-extension", "false")
		url = l.MustLaunch()
	}

	// Conectamos Go-Rod al navegador
	browser := rod.New().ControlURL(url).MustConnect()

	return &Manager{
		Browser: browser,
		DataDir: userDataDir,
	}
}

// Close cierra el navegador
func (m *Manager) Close() {
	if m.Browser != nil {
		m.Browser.MustClose()
	}
}

// ManualLogin abre una página y espera a que el usuario cierre el navegador
// Esto permite al usuario interactuar libremente para loguearse
func (m *Manager) ManualLogin() {
	// Navegar primero a Google home para "calentar" la sesión
	// Sin stealth, usamos el navegador tal cual (confiando en las flags y en que es el binario del sistema)
	page := m.Browser.MustPage(URLGoogleHome)

	page.MustNavigate(URLGoogleLogin)

	fmt.Println(i18n.T("browser_nav_open"))
	fmt.Println(i18n.T("browser_nav_close"))

	page.MustWaitOpen() // Espera a que la página se cargue

	// Bloquea la ejecución hasta que se cierre el navegador
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		if _, err := m.Browser.Pages(); err != nil {
			break
		}
	}
}

// VerifySession comprueba si las cookies actuales permiten acceder a Google Photos
func (m *Manager) VerifySession() bool {
	fmt.Println(i18n.T("verifying_session"))
	// Vamos a photos.google.com
	page := m.Browser.MustPage(URLGooglePhotos)

	// Esperamos a que la página se estabilice (redirecciones, carga de scripts)
	// Usamos MustWaitLoad con timeout porque MustWaitStable se cuelga con el tráfico de fondo de Google Photos
	page.Timeout(15 * time.Second).MustWaitLoad()

	// Obtenemos la URL final
	url := page.MustInfo().URL

	// Si la URL sigue siendo photos.google.com, estamos logueados.
	// Si nos redirige a accounts.google.com o about.google, falló.
	return strings.Contains(url, "photos.google.com")
}

// RequestTakeout automatiza la solicitud de un backup de Google Photos en Takeout
func (m *Manager) RequestTakeout() error {
	fmt.Println(i18n.T("navigating_takeout"))
	// Forzamos el idioma inglés (hl=en) para que los selectores por aria-label funcionen siempre
	page := m.Browser.MustPage(URLTakeoutSettings)
	page.MustWaitLoad()

	// Esperar a que el botón "Deselect all" esté visible y hacer clic
	fmt.Println(i18n.T("deselecting_products"))
	// Usamos selectores robustos basados en atributos que Google usa internamente
	page.MustElement(`[aria-label="Deselect all"]`).MustClick()
	time.Sleep(1 * time.Second) // Pequeña pausa para que la UI reaccione

	// Seleccionar solo Google Photos
	fmt.Println(i18n.T("selecting_photos"))

	// Estrategia robusta: Usamos XPath para buscar el texto EXACTO "Google Photos".
	// normalize-space() elimina espacios extra y evita coincidencias parciales en descripciones de otros productos.
	productLabel := page.MustElementX(`//div[normalize-space(text())="Google Photos"]`)

	// Subimos niveles hasta encontrar el contenedor del producto que tiene el checkbox
	found := false
	parent := productLabel
	for i := 0; i < 10; i++ { // Intentamos hasta 10 niveles hacia arriba
		var err error
		parent, err = parent.Parent()
		if err != nil {
			break
		}
		if has, _, _ := parent.Has(`input[type="checkbox"]`); has {
			parent.MustElement(`input[type="checkbox"]`).MustClick()
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no se pudo encontrar el checkbox de Google Photos")
	}

	// Ir al siguiente paso
	fmt.Println(i18n.T("next_step"))
	page.MustElement(`button[aria-label="Next step"]`).MustClick()

	// Esperar a que la sección de creación de exportación cargue
	page.MustWaitLoad()

	// Seleccionar 50GB para reducir número de archivos (menos ZIPs que descargar)
	fmt.Println(i18n.T("config_size"))
	// Abrir menú de tamaño
	page.MustElement(`div[aria-label="File size select"]`).MustClick()
	time.Sleep(500 * time.Millisecond)
	// Seleccionar opción de 50 GB
	page.MustElementR("li", "50 GB").MustClick()
	time.Sleep(500 * time.Millisecond)

	// Crear la exportación
	fmt.Println(i18n.T("creating_export"))
	page.MustElementR("button", "Create export").MustClick()

	// Esperar a la página de confirmación
	fmt.Println(i18n.T("waiting_confirmation"))
	page.MustWaitNavigation()

	return nil
}

// ExportStatus representa el estado de una exportación en Takeout
type ExportStatus struct {
	InProgress    bool
	Completed     bool
	DownloadLinks []string
	CreateTime    time.Time
	CreatedAt     time.Time
	ID            string
	StatusText    string // Texto crudo del estado (ej: "Completo", "Cancelado")
}

// CheckExportStatus comprueba si hay exportaciones activas o listas para descargar
func (m *Manager) CheckExportStatus() ([]ExportStatus, error) {
	fmt.Println(i18n.T("checking_status"))
	page := m.Browser.MustPage(URLTakeoutManage)
	page.MustWaitLoad()

	var statuses []ExportStatus

	// 1. Comprobar si hay exportación en curso
	// Buscamos el texto "Export in progress..." o "Your files are currently being prepared"
	// Basado en el HTML proporcionado: <div class="PEJjGd">Export in progress...</div>
	// y <div class="yrYG6">Your files are currently being prepared</div>

	// Usamos HasR para buscar texto visible, es más robusto que clases ofuscadas
	// Nota: Buscamos "Export in progress" o "files are currently being prepared"
	// BUG FIX: Si hay varias, el texto es "Exports in progress..." (plural).
	// En lugar de depender del texto del título, buscamos directamente los elementos de progreso.
	if elements, err := page.Elements(`div[data-in-progress="true"]`); err == nil && len(elements) > 0 {
		for _, el := range elements {
			// Verificar si es de Google Photos
			text, _ := el.Text()
			if strings.Contains(text, "Google Photos") {
				current := ExportStatus{InProgress: true}
				fmt.Println(i18n.T("export_in_progress"))

				if attr, err := el.Attribute("data-archive-id"); err == nil && attr != nil {
					current.ID = *attr
				}

				// Intentar extraer la fecha de creación dentro del elemento de progreso
				if createdEl, err := el.ElementR("div", "Created:"); err == nil {
					text, _ := createdEl.Text()
					if idx := strings.Index(text, "Created:"); idx != -1 {
						dateStr := text[idx+len("Created:"):]
						if i := strings.IndexAny(dateStr, "\r\n"); i != -1 {
							dateStr = dateStr[:i]
						}
						dateStr = strings.ReplaceAll(dateStr, "\u202f", " ")
						dateStr = strings.TrimSpace(dateStr)
						layout := "January 2, 2006, 3:04 PM"
						if t, err := time.Parse(layout, dateStr); err == nil {
							current.CreatedAt = t
						}
					}
				}
				statuses = append(statuses, current)
			} else {
				fmt.Println(i18n.T("ignoring_other"))
			}
		}
	}

	// 2. Iterar sobre la lista de exportaciones pasadas (Completadas, Canceladas, etc.)
	// Buscamos la lista ul[jsname="archivelist"] y sus elementos li
	if list, err := page.Element(`ul[jsname="archivelist"]`); err == nil {
		items, _ := list.Elements("li")
		for _, item := range items {
			// Verificar si es de Google Photos
			text, _ := item.Text()
			if !strings.Contains(text, "Google Photos") {
				continue
			}

			var st ExportStatus

			// Extraer ID del enlace
			// <a href="./manage/archive/ID_AQUI" ...>
			if link, err := item.Element("a"); err == nil {
				href, _ := link.Attribute("href")
				if href != nil {
					parts := strings.Split(*href, "/")
					if len(parts) > 0 {
						st.ID = parts[len(parts)-1]
					}
				}
			}

			// Extraer Estado (Texto visible)
			// <p class="BXHFQ">Completo</p> o <p class="BXHFQ">Cancelado</p>
			// Nota: Al forzar ?hl=en, esperamos "Complete", "Cancelled", etc.
			if statusEl, err := item.Element(`p.BXHFQ`); err == nil {
				text, _ := statusEl.Text()
				st.StatusText = text
				if strings.Contains(text, "Complete") {
					st.Completed = true
				}
			}

			// TODO: Extraer fecha de creación si es necesario (está en un div dentro del li)

			statuses = append(statuses, st)
		}
	}

	return statuses, nil
}

// CancelExport cancela una exportación en curso
func (m *Manager) CancelExport() error {
	fmt.Println(i18n.T("cancelling_stale"))
	page := m.Browser.MustPage(URLTakeoutManage)
	page.MustWaitLoad()

	// Buscar botón "Cancel export"
	// HTML: <button ... aria-label="Cancel export..."><span ...>Cancel export</span></button>
	btn, err := page.ElementR("button", "Cancel export")
	if err != nil {
		return fmt.Errorf("no se encontró el botón de cancelar exportación")
	}

	btn.MustClick()
	time.Sleep(2 * time.Second) // Esperar a que la UI se actualice
	fmt.Println(i18n.T("cancel_sent"))
	return nil
}

// GetDownloadList extracts the list of files to be downloaded from the export page
func (m *Manager) GetDownloadList(id string) ([]registry.DownloadFile, error) {
	fmt.Printf(i18n.T("download_start")+"\n", id)

	url := fmt.Sprintf(URLTakeoutArchive, id)
	page := m.Browser.MustPage(url)
	page.MustWaitLoad()

	var files []registry.DownloadFile

	// Find all download buttons/containers
	// We look for the structure:
	// <div class="xsr7od"><div>SIZE</div>...</div> ... <div ...><a ... aria-label="Download part X of Y"></a>

	// Better strategy: Find all download links first, then deduce part number from them
	// Buttons have aria-label="Download part X of Y" or "Download again part X of Y"
	// Also might be just "Download" if single file.

	// Use generic selector for download links
	// Link href usually contains "takeout/download"
	links, err := page.Elements(`a[href*="takeout/download"]`)
	if err != nil {
		return nil, fmt.Errorf("failed to find download links: %v", err)
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("no download links found (expired?)")
	}

	for _, link := range links {
		// Get aria-label to identify part
		ariaLabel, err := link.Attribute("aria-label")
		if err != nil || ariaLabel == nil {
			continue
		}
		// Filter: Must be a "Download" link, not "See report"
		// "Download part X of Y" or "Download" or "Download again..."
		if !strings.Contains(*ariaLabel, "Download") {
			continue
		}

		// Determinar número de parte basado en el orden de aparición de enlaces válidos
		// Esto asume que aparecen en orden 1..N
		partNum := len(files) + 1

		f := registry.DownloadFile{
			PartNumber: partNum,
			Status:     "pending",
		}

		files = append(files, f)
	}

	return files, nil
}

var ErrQuotaExceeded = fmt.Errorf("download quota exceeded (5 attempts limit)")

// DownloadFiles downloads files in parallel (fire-and-watch) to avoid auth timeout
func (m *Manager) DownloadFiles(id string, files []registry.DownloadFile, destDir string, password string, updateStatus func(int, registry.DownloadFile)) error {
	url := fmt.Sprintf(URLTakeoutArchive, id)
	fmt.Printf("⏳ Navigating to: %s\n", url)

	page := m.Browser.MustPage(url)
	// Removed redundant page.MustWaitLoad() which might hang on some pages.
	// Instead, we wait for the container that holds the download list.
	fmt.Println("⏳ Waiting for page content...")
	container := page.MustElement(`[data-export-type]`) // Waits for the main container

	// Check for Quota Exceeded directly on the container attribute
	fmt.Println("🔍 Checking for quota limit...")
	if val, err := container.Attribute("data-download-quota-exceeded"); err == nil && val != nil && *val == "true" {
		return ErrQuotaExceeded
	}

	// 1. Identification
	fmt.Println("🔍 Identifying pending files...")
	var pendingIndices []int
	for i, f := range files {
		if f.Status != "completed" {
			pendingIndices = append(pendingIndices, i)
		}
	}

	if len(pendingIndices) == 0 {
		fmt.Println("✅ No pending files to download.")
		return nil
	}
	fmt.Printf("📋 Found %d pending files.\n", len(pendingIndices))

	// Set download behavior ONCE for the page
	proto.PageSetDownloadBehavior{
		Behavior:     proto.PageSetDownloadBehaviorBehaviorAllow,
		DownloadPath: destDir,
	}.Call(page)

	// Identify buttons
	fmt.Println("🔍 Locating download buttons...")
	links, err := page.Elements(`a[href*="takeout/download"]`)
	if err != nil {
		return fmt.Errorf("failed to find links: %w", err)
	}

	// Filter valid links (same logic as GetDownloadList)
	// AND rewrite hrefs to force English
	var validLinks rod.Elements
	for _, link := range links {
		attr, _ := link.Attribute("aria-label")
		if attr != nil && strings.Contains(*attr, "Download") {
			// Rewrite href to include hl=en
			if href, err := link.Attribute("href"); err == nil && href != nil {
				newHref := *href
				if !strings.Contains(newHref, "hl=en") {
					if strings.Contains(newHref, "?") {
						newHref += "&hl=en"
					} else {
						newHref += "?hl=en"
					}
					_, _ = link.Eval(`(el, val) => el.setAttribute("href", val)`, newHref)
				}
			}
			validLinks = append(validLinks, link)
		}
	}
	fmt.Printf("found %d valid 'Download' buttons\n", len(validLinks))

	// 3. Setup Channels for Coordination
	// We need to stop waiting if an error occurs (like Quota Exceeded)
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// 4. Setup Global Listener
	guidMap := make(map[string]int)
	completedCount := 0
	totalToDownload := len(pendingIndices)

	// wait() blocks until the callbacks return true
	// We run it in a goroutine so we can interrupt it if we detect Quota Exceeded
	go func() {
		defer close(doneChan)
		page.EachEvent(
			func(e *proto.PageDownloadWillBegin) bool {
				idx := -1
				// Match by PartNumber suffix in filename?
				// e.SuggestedFilename: "takeout-...-003.zip"
				// files[i].Filename might be empty or "takeout-...-003.zip"
				// We need to map SuggestedFilename to our files list
				// Simple heuristic: Extract part number from SuggestedFilename
				// format: ...-NNN.zip
				// We look for files with matching PartNumber

				// Let's iterate files to find matching part number
				// This assumes standard Takeout naming: match the NNN part
				// e.g. "takeout-20240201-001.zip"

				// Helper to extract NNN
				parts := strings.Split(strings.TrimSuffix(e.SuggestedFilename, ".zip"), "-")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1] // "001"
					// Wait, Sscanf is tricky. usage: fmt.Sscanf("001", "%d", &num)
					var pNum int
					if _, err := fmt.Sscanf(lastPart, "%d", &pNum); err == nil {
						// Find file with PartNumber == pNum
						for i, f := range files {
							if f.PartNumber == pNum {
								idx = i
								break
							}
						}
					}
				}

				if idx != -1 {
					guidMap[e.GUID] = idx
					files[idx].Filename = e.SuggestedFilename
					files[idx].Status = "downloading"
					files[idx].URL = e.URL
					fmt.Printf("\n     ... Started: %s\n", e.SuggestedFilename)
					updateStatus(idx, files[idx])
				} else {
					fmt.Printf("\n⚠️  Unknown download started: %s\n", e.SuggestedFilename)
				}
				return false
			},
			func(e *proto.PageDownloadProgress) bool {
				if idx, ok := guidMap[e.GUID]; ok {
					if e.State == proto.PageDownloadProgressStateCompleted {
						files[idx].Status = "completed"
						files[idx].DownloadedBytes = int64(e.ReceivedBytes)
						files[idx].SizeBytes = int64(e.TotalBytes)
						updateStatus(idx, files[idx])
						completedCount++

						if completedCount >= totalToDownload {
							return true
						}
					} else if e.State == proto.PageDownloadProgressStateCanceled {
						files[idx].Status = "failed"
						updateStatus(idx, files[idx])
						completedCount++
						if completedCount >= totalToDownload {
							return true
						}
					} else {
						// Active
						files[idx].DownloadedBytes = int64(e.ReceivedBytes)
						files[idx].SizeBytes = int64(e.TotalBytes)
						updateStatus(idx, files[idx])
					}
				}
				return false
			},
		)()
	}()

	// 3. Fire Clicks in Background
	go func() {
		fmt.Println("🚀 Firing download requests...")
		for _, fileIdx := range pendingIndices {
			partNum := files[fileIdx].PartNumber
			if partNum > len(validLinks) {
				fmt.Printf("❌ Link not found for part %d\n", partNum)
				continue
			}

			btn := validLinks[partNum-1]

			// Clean partials first
			globPattern := filepath.Join(destDir, fmt.Sprintf("*-%03d.zip.crdownload", partNum))
			if partials, _ := filepath.Glob(globPattern); len(partials) > 0 {
				for _, p := range partials {
					os.Remove(p)
				}
			}

			// Click
			// We accept that this might trigger auth.
			// If we do strict parallel, we might race on auth input.
			// But usually auth is once per session.
			// We add a small delay to be gentle and allow auth check to potentially appear
			// If auth appears, it blocks.

			// Check for Auth Prompt BEFORE clicking? No, it appears AFTER.
			// If Auth appears, we should probably handle it.
			// But handling 17 auths?
			// Hopefully only the first one triggers it.

			// We use a retry mechanism for the click?
			if err := btn.Click(proto.InputMouseButtonLeft, 1); err != nil {
				fmt.Printf("❌ Failed to click part %d: %v\n", partNum, err)

				// Check for Quota Exceeded in URL (Strongest Signal)
				if info, err := page.Info(); err == nil && strings.Contains(info.URL, "quotaExceeded=true") {
					fmt.Println("⛔ Detected Quota Exceeded via URL parameter.")
					errChan <- ErrQuotaExceeded
					return
				}

				// Check for Quota Exceeded in Page Content (English or Spanish)
				// Spanish: "No puedes volver a descargar"
				html, _ := page.HTML()
				if strings.Contains(html, "data-download-quota-exceeded=\"true\"") ||
					strings.Contains(html, "No puedes volver a descargar") {
					fmt.Println("⛔ Detected Quota Exceeded via Page Content.")
					errChan <- ErrQuotaExceeded
					return
				}

				files[fileIdx].Status = "failed"
				updateStatus(fileIdx, files[fileIdx])
				// We still continue to trigger others? Yes.
			}

			// Quick auth check
			// m.handleAuth(password) // This looks for input on the page.

			time.Sleep(2 * time.Second) // Small delay between fires
		}
		fmt.Println("✅ Download requests firing phase completed.") // Changed message
	}()

	// 4. Wait for all to finish
	// 6. Wait for Completion or Error
	select {
	case <-doneChan:
		return nil
	case err := <-errChan:
		return err
	}
}

// downloadSingleFile is deprecated/removed in favor of parallel logic inside DownloadFiles
// We keep handleAuth helper

func (m *Manager) handleAuth(password string) {
	if password == "" {
		return
	}
	// Check for password input
	// input[type="password"]
	// This might happen in the same page (modal) or new page.
	// Ideally check current page.
	page := m.Browser.MustPages().First()
	if el, err := page.Element(`input[type="password"]`); err == nil {
		fmt.Println("🔑 Auth prompt detected. Attempting to enter password...")
		el.MustInput(password)
		time.Sleep(500 * time.Millisecond)
		// Send Enter key
		el.MustInput("\n")
	}
}

// Deprecated: Use DownloadFiles
func (m *Manager) DownloadExport(id string, destDir string) (int, string, error) {
	// Wrapping new logic for compatibility if needed, else delete.
	return 0, "", fmt.Errorf("use DownloadFiles instead")
}
