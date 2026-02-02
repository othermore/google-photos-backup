package browser

import (
	"fmt"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	urlPkg "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
		logger.Debug(i18n.T("browser_system"), path)
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
		logger.Info(i18n.T("browser_download_fail"))
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

	// 🕵️♂️ HIJACKER: Enforce English (hl=en) on all Takeout requests
	// This intercepts every request to takeout.google.com and appends hl=en if missing.
	router := browser.HijackRequests()
	router.MustAdd("*.google.com*", func(ctx *rod.Hijack) {
		// Only touch requests that are navigational or documents? No, all is safer for consistency.
		// But mostly we care about the main frame.
		// To be safe and avoid breaking APIs, let's only target takeout URLs for now.
		currentURL := ctx.Request.URL().String()
		if strings.Contains(currentURL, "takeout.google.com") {
			u, err := urlPkg.Parse(currentURL)
			if err == nil {
				q := u.Query()
				if q.Get("hl") != "en" {
					q.Set("hl", "en")
					u.RawQuery = q.Encode()
					// Modify the URL being requested via ContinueRequest options
					ctx.ContinueRequest(&proto.FetchContinueRequest{
						URL: u.String(),
					})
					return
				}
			}
		}
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go router.Run()

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

// RequestTakeout navigates to Takeout and requests a new export
func (m *Manager) RequestTakeout(mode string) error {
	// TODO: Use mode to select delivery method (Email vs Drive)
	// For now, default to Email (Direct Download)
	logger.Debug("🚀 Requesting new export (Mode: %s)...", mode)

	logger.Debug(i18n.T("navigating_takeout"))
	// Forzamos el idioma inglés (hl=en) para que los selectores por aria-label funcionen siempre
	page := m.Browser.MustPage(URLTakeoutSettings)
	page.MustWaitLoad()

	// Esperar a que el botón "Deselect all" esté visible y hacer clic
	logger.Debug(i18n.T("deselecting_products"))
	// Usamos selectores robustos basados en atributos que Google usa internamente
	page.MustElement(`[aria-label="Deselect all"]`).MustClick()
	time.Sleep(1 * time.Second) // Pequeña pausa para que la UI reaccione

	// Seleccionar solo Google Photos
	logger.Debug(i18n.T("selecting_photos"))

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
	logger.Debug(i18n.T("next_step"))
	page.MustElement(`button[aria-label="Next step"]`).MustClick()

	// Esperar a que la sección de creación de exportación cargue
	page.MustWaitLoad()

	// Seleccionar 50GB para reducir número de archivos (menos ZIPs que descargar)
	logger.Debug(i18n.T("config_size"))
	// Abrir menú de tamaño
	page.MustElement(`div[aria-label="File size select"]`).MustClick()
	time.Sleep(500 * time.Millisecond)
	// Seleccionar opción de 50 GB
	page.MustElementR("li", "50 GB").MustClick()
	time.Sleep(500 * time.Millisecond)

	// Crear la exportación
	logger.Debug(i18n.T("creating_export"))

	// Setup navigation wait BEFORE clicking to avoid race condition
	wait := page.MustWaitNavigation()
	page.MustElementR("button", "Create export").MustClick()

	// Wait for navigation to complete
	logger.Debug(i18n.T("waiting_confirmation"))
	wait()

	// Ensure we are on the Manage page
	// Sometimes it redirects to /settings/takeout/custom/..., then eventually to /manage
	// We'll wait for the URL to contain "manage"
	logger.Debug("Waiting for redirect to Manage page...")
	err := page.WaitElementsMoreThan(`div[data-in-progress="true"], button[aria-label="Cancel export"]`, 0)
	// Or just wait for URL
	if err != nil {
		// Fallback: Check URL loop
		for i := 0; i < 10; i++ {
			if strings.Contains(page.MustInfo().URL, "manage") {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	time.Sleep(2 * time.Second) // Extra buffer for backend propagation

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
	// Strategy A: data-in-progress attribute (official/clean way)
	if elements, err := page.Elements(`div[data-in-progress="true"]`); err == nil && len(elements) > 0 {
		for _, el := range elements {
			// Verificar si es de Google Photos
			text, _ := el.Text()
			if strings.Contains(text, "Google Photos") {
				current := ExportStatus{InProgress: true}
				logger.Debug(i18n.T("export_in_progress"))

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

	// Strategy B: Fallback - Look for "Cancel export" buttons (UI way)
	// If Strategy A found nothing, this catches cases where attributes changed
	if len(statuses) == 0 {
		// Buscamos botones de cancelar, que solo aparecen en exportaciones activas
		cancelButtons, err := page.Elements(`button[aria-label="Cancel export"]`)
		if err == nil && len(cancelButtons) > 0 {
			// Hay al menos una en progreso. Intentamos deducir si es de Google Photos.
			logger.Info("⚠️  Detected 'Cancel export' button. Assuming export in progress.")
			statuses = append(statuses, ExportStatus{InProgress: true, ID: "unknown-pending"})
		} else {
			// Texto plano como último recurso (English ONLY)
			bodyText, _ := page.Element("body")
			text, _ := bodyText.Text()
			if strings.Contains(text, "being prepared") || strings.Contains(text, "Export in progress") {
				logger.Info("⚠️  Detected in-progress text on page. Waiting.")
				statuses = append(statuses, ExportStatus{InProgress: true, ID: "unknown-pending"})
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
	logger.Info(i18n.T("cancel_sent"))
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
func (m *Manager) DownloadFiles(id string, files []registry.DownloadFile, destDir string, updateStatus func(int, registry.DownloadFile)) error {
	url := fmt.Sprintf(URLTakeoutArchive, id)
	logger.Debug("⏳ Navigating to: %s", url)

	page := m.Browser.MustPage(url)
	fmt.Println("⏳ Waiting for page content...")
	container := page.MustElement(`[data-export-type]`)

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
		logger.Info("✅ No pending files to download.")
		return nil
	}
	logger.Info("📋 Found %d pending files. Scraping URLs...", len(pendingIndices))

	// 2. Scrape URLs Upfront (Robustness Fix)
	// We extract map[PartNumber]URL to allow closing/ignoring the main page later
	partMap := make(map[int]string)

	// Mutex for safe concurrent access during scraping and downloading
	var mu sync.Mutex

	// Get base URL for resolution
	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info: %v", err)
	}
	baseURL, err := urlPkg.Parse(info.URL)
	if err != nil {
		fmt.Printf("⚠️  Failed to parse base URL %s: %v\n", info.URL, err)
		// Proceed but relative links might fail
	}

	// Scrape elements that have the download URI and Size
	// DOM: <div ... data-download-uri="..." data-size="...">
	links, err := page.Elements(`[data-download-uri]`)
	if err == nil {
		for _, el := range links {
			href, _ := el.Attribute("data-download-uri") // e.g. "takeout/download?..." (relative)
			if href != nil {
				// Resolve URL
				finalURL := *href
				if strings.HasPrefix(*href, "takeout/") {
					finalURL = "/" + *href
				}
				if baseURL != nil {
					if ref, err := urlPkg.Parse(finalURL); err == nil {
						finalURL = baseURL.ResolveReference(ref).String()
					}
				}

				// Extract PartNumber from the URL/Element
				// URL: ...&i=0&... (i=0 is part 1)
				idx := strings.Index(finalURL, "&i=")
				if idx == -1 {
					idx = strings.Index(finalURL, "?i=")
				}

				if idx != -1 {
					remaining := finalURL[idx+3:] // after "i="
					var iParam int
					if _, err := fmt.Sscanf(remaining, "%d", &iParam); err == nil {
						// Check if the container or link inside is actually "See report" (Not a file part)
						isReport := false
						if links, err := el.Elements("a"); err == nil {
							for _, l := range links {
								if lbl, err := l.Attribute("aria-label"); err == nil && lbl != nil {
									if strings.Contains(*lbl, "See report") || strings.Contains(*lbl, "Show details") {
										isReport = true
										break
									}
								}
							}
						}

						if !isReport {
							partNum := iParam + 1
							partMap[partNum] = finalURL

							// Also scrape SIZE if available
							if sizeStr, err := el.Attribute("data-size"); err == nil && sizeStr != nil {
								var sizeBytes int64
								if _, err := fmt.Sscanf(*sizeStr, "%d", &sizeBytes); err == nil {
									// Update file size in our list
									mu.Lock()
									for idx, f := range files {
										if f.PartNumber == partNum {
											files[idx].SizeBytes = sizeBytes
											updateStatus(idx, files[idx])
											break
										}
									}
									mu.Unlock()
								}
							}
						}
					}
				}
			}
		}
	}
	logger.Debug("📋 Scraped %d valid download links.", len(partMap))

	// 3. Setup Channels for Coordination
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// 4. Setup Global Listener (Browser Level)
	guidMap := make(map[string]int)
	processedCount := 0 // successes + failures
	totalToDownload := len(pendingIndices)
	allProcessedChan := make(chan struct{}, 1)

	checkDone := func() bool {
		mu.Lock()
		defer mu.Unlock()
		logger.Debug("[DEBUG] checkDone: processed %d / %d", processedCount, totalToDownload)
		if processedCount >= totalToDownload {
			select {
			case allProcessedChan <- struct{}{}:
			default:
			}
			return true
		}
		return false
	}

	// Wait loop using Browser.EachEvent
	go func() {
		defer close(doneChan)
		// We use m.Browser, not page
		wait := m.Browser.EachEvent(
			func(e *proto.PageDownloadWillBegin) bool {
				idx := -1
				// Match by PartNumber suffix in filename?
				parts := strings.Split(strings.TrimSuffix(e.SuggestedFilename, ".zip"), "-")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
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
					mu.Lock()
					guidMap[e.GUID] = idx
					files[idx].Filename = e.SuggestedFilename
					files[idx].Status = "downloading"
					files[idx].URL = e.URL
					mu.Unlock()
					fmt.Printf("\n     ... Started: %s\n", e.SuggestedFilename)
					updateStatus(idx, files[idx])
				} else {
					fmt.Printf("\n⚠️  Unknown download started: %s\n", e.SuggestedFilename)
				}
				return false
			},
			func(e *proto.PageDownloadProgress) bool {
				mu.Lock()
				idx, ok := guidMap[e.GUID]
				mu.Unlock()

				if ok {
					if e.State == proto.PageDownloadProgressStateCompleted {
						mu.Lock()
						files[idx].Status = "completed"
						files[idx].DownloadedBytes = int64(e.ReceivedBytes)
						files[idx].SizeBytes = int64(e.TotalBytes)
						processedCount++
						mu.Unlock()

						// Attempt to move file
						homeDir, err := os.UserHomeDir()
						if err == nil {
							downloadPath := filepath.Join(homeDir, "Downloads", files[idx].Filename)
							targetPath := filepath.Join(destDir, files[idx].Filename)

							// Check if file exists in Downloads
							if _, err := os.Stat(downloadPath); err == nil {
								if err := os.Rename(downloadPath, targetPath); err != nil {
									fmt.Printf("\n⚠️  Failed to move file from %s to %s: %v\n", downloadPath, targetPath, err)
								}
							}
						}

						updateStatus(idx, files[idx])
						if checkDone() {
							return true
						}
					} else if e.State == proto.PageDownloadProgressStateCanceled {
						mu.Lock()
						files[idx].Status = "failed"
						processedCount++
						mu.Unlock()

						updateStatus(idx, files[idx])
						if checkDone() {
							return true
						}
					} else {
						// Active
						mu.Lock()
						files[idx].DownloadedBytes = int64(e.ReceivedBytes)
						files[idx].SizeBytes = int64(e.TotalBytes)
						mu.Unlock()
						updateStatus(idx, files[idx])
					}
				}
				return false
			},
		)
		wait()
	}()

	// 5. Fire Downloads via Button Click (Sequential)
	// We run this in a goroutine so the main thread can block on the done/error channels
	// Helper to track active downloads
	getActive := func() int {
		mu.Lock()
		defer mu.Unlock()
		c := 0
		for _, f := range files {
			if f.Status == "downloading" {
				c++
			}
		}
		return c
	}

	// 5. Fire Downloads via Button Click (Sequential with Throttling)
	// We run this in a goroutine so the main thread can block on the done/error channels
	go func() {
		logger.Debug("🚀 Firing download requests via Button Click (Robust JS)...")

		// Helper to click a file and check for errors (Quota/Modal)
		clickFileWithCheck := func(fileIdx int) bool {
			pNum := files[fileIdx].PartNumber

			// 1. Ensure we are on the correct page (recover from Auth redirects) and LANGUAGE is English
			currentInfo, err := page.Info()
			currentURL := "unknown"
			if currentInfo != nil {
				currentURL = currentInfo.URL
			}

			// Validate URL path and Query param (hl=en)
			// If missing hl=en, we reload to ensure English text for error detection
			if err != nil || !strings.Contains(currentURL, "takeout.google.com/manage/archive") || !strings.Contains(currentURL, "hl=en") {
				logger.Debug("⚠️  Page context/lang lost (URL: %s). Navigating back to list with hl=en...", currentURL)

				// Ensure target URL has hl=en
				targetURL := url
				if !strings.Contains(targetURL, "hl=en") {
					if strings.Contains(targetURL, "?") {
						targetURL += "&hl=en"
					} else {
						targetURL += "?hl=en"
					}
				}

				page.MustNavigate(targetURL)
				page.MustWaitLoad()
				time.Sleep(5 * time.Second) // Stabilize
			}

			logger.Debug("   ... Clicking button for part %d...", pNum)

			// Clean partials
			globPattern := filepath.Join(destDir, fmt.Sprintf("*-%03d.zip.crdownload", pNum))
			if partials, _ := filepath.Glob(globPattern); len(partials) > 0 {
				for _, p := range partials {
					os.Remove(p)
				}
			}

			// 2. Click via JS
			jsScript := fmt.Sprintf(`() => {
				const links = Array.from(document.querySelectorAll('a[href*="takeout/download"]'));
				const target = links.find(l => {
					if (!l.href.includes("i=%d")) return false;
					const label = l.getAttribute("aria-label") || "";
					if (label.includes("See report") || label.includes("Show details")) return false;
					return true;
				});
				if (target) {
					target.click();
					return true;
				}
				return false;
			}`, pNum-1)

			res, err := page.Eval(jsScript)
			if err != nil {
				fmt.Printf("❌ JS Execution failed for part %d: %v\n", pNum, err)
				return false
			}

			// Check if click was successful
			if res != nil && res.Value.Bool() {
				// Check for Quota Modal or URL Redirect error immediately after click
				time.Sleep(2 * time.Second) // Give modal time to appear or redirect to happen

				// 3. Check URL for Error Flags (Robust against Language)
				if info, err := page.Info(); err == nil {
					postClickURL := info.URL
					if strings.Contains(postClickURL, "quotaExceeded=true") {
						logger.Error("⛔ Quota Exceeded detected via URL Param for part %d! Aborting export.", pNum)
						mu.Lock()
						files[fileIdx].Status = "failed"
						mu.Unlock()

						// Signal Fatal Error to Main Thread
						select {
						case errChan <- ErrQuotaExceeded:
						default:
						}

						return false
					}

					// User reported redirects to Spanish pages without hl=en
					// If we were redirected to a new page (e.g. error page) and lost hl=en, force reload
					// But only if we are still on Takeout
					if strings.Contains(postClickURL, "takeout.google.com") && !strings.Contains(postClickURL, "hl=en") {
						logger.Debug("⚠️  Redirected to non-English page (URL: %s). Reloading with hl=en...", postClickURL)
						newURL := postClickURL
						if strings.Contains(newURL, "?") {
							newURL += "&hl=en"
						} else {
							newURL += "?hl=en"
						}
						page.MustNavigate(newURL)
						page.MustWaitLoad()
						time.Sleep(2 * time.Second)
					}
				}

				// Check for the specific modal DOM user provided
				// role="dialog" and text "You can't download this file again"
				// We search for the header text h2[id="c1"] or just headers
				// User validated that we should rely on English by ensuring hl=en
				modalScript := `() => {
					const dialogs = Array.from(document.querySelectorAll('div[role="dialog"]'));
					for (const d of dialogs) {
						const text = d.innerText.toLowerCase();
						if (text.includes("you can't download this file again") || 
							text.includes("maximum number of times") ||
							text.includes("quota exceeded")) {
							return true;
						}
					}
					return false;
				}`
				modalRes, _ := page.Eval(modalScript)
				if modalRes != nil && modalRes.Value.Bool() {
					logger.Error("⛔ Quota Exceeded detected (Modal) for part %d! Aborting export.", pNum)
					// Handle Fatal Error
					mu.Lock()
					files[fileIdx].Status = "failed"
					mu.Unlock()

					// Signal Fatal Error to Main Thread
					select {
					case errChan <- ErrQuotaExceeded:
					default:
					}

					return false
				}

				return true
			}
			return false
		}

		// PHASE 1: WARM-UP (First file only)
		if len(pendingIndices) > 0 {
			firstIdx := pendingIndices[0]
			partNum := files[firstIdx].PartNumber
			logger.Info("🔑 Warm-up: Starting first download (Part %d) to validate session...", partNum)

			if clickFileWithCheck(firstIdx) {
				// Wait for it to become "downloading"
				logger.Info("⏳ Waiting for download to start (Check browser for Passkey)...")
				timeout := time.After(2 * time.Minute)
				ticker := time.NewTicker(2 * time.Second)
				started := false

				for !started {
					select {
					case <-timeout:
						logger.Info("⚠️  Timed out waiting for first download. Proceeding anyway...")
						started = true
					case <-ticker.C:
						mu.Lock()
						status := files[firstIdx].Status
						if status == "downloading" || status == "completed" {
							started = true
						} else if status == "failed" {
							// Check if it failed due to our logic above
							started = true
						}
						mu.Unlock()
					}
				}
				ticker.Stop()
			}
		}

		// PHASE 2: REST OF FILES (Throttled)
		// Start from index 1 (since 0 is done in warm-up)
		startLoop := 1
		if len(pendingIndices) <= 1 {
			startLoop = len(pendingIndices) // nothing left
		}

		maxConcurrent := 2 // Limit parallel downloads

		for i := startLoop; i < len(pendingIndices); i++ {
			fileIdx := pendingIndices[i]
			partNum := files[fileIdx].PartNumber

			// Throttle: Wait if too many active
			for getActive() >= maxConcurrent {
				logger.Debug("⏳ Max concurrent downloads (%d) reached. Waiting...", maxConcurrent)
				time.Sleep(10 * time.Second)
			}

			// Use the helper logic
			success := clickFileWithCheck(fileIdx)

			if success {
				logger.Debug("✅ Click successful for part %d", partNum)
				time.Sleep(15 * time.Second) // Slow down
			} else {
				logger.Debug("⚠️  Click failed (JS false or Quota detected) for part %d. Skipping.", partNum)
				mu.Lock()
				files[fileIdx].Status = "failed"
				processedCount++
				mu.Unlock()
				updateStatus(fileIdx, files[fileIdx])
				checkDone()
			}
		}
		logger.Debug("✅ Download requests firing phase completed.")

		if checkDone() {
			// Trigger a check just in case
		}
	}()

	// 6. Wait for Completion or Error
	select {
	case <-doneChan: // from EachEvent (wait returns)
		return nil
	case <-allProcessedChan: // all files accounted for
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

// ParseSize parses human readable size string (e.g. "50 GB", "10.5 MB") into bytes
func ParseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	var val float64
	var unit string
	// Try parsing with unit
	if _, err := fmt.Sscanf(s, "%f %s", &val, &unit); err != nil {
		// Fallback for just numbers?
		return 0
	}

	multiplier := int64(1)
	switch unit {
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	}
	return int64(val * float64(multiplier))
}
