package browser

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
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
		Headless(headless).       // Sin interfaz gráfica si es true
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
	page := m.Browser.MustPage("https://www.google.com")

	page.MustNavigate("https://accounts.google.com")

	fmt.Println("ℹ️  Navegador abierto. Por favor, inicia sesión en tu cuenta de Google.")
	fmt.Println("ℹ️  Cuando hayas terminado, simplemente cierra la ventana del navegador.")

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
	fmt.Println("🔍 Verificando sesión en segundo plano...")
	// Vamos a photos.google.com
	page := m.Browser.MustPage("https://photos.google.com")

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
	fmt.Println("🚀 Navegando a Google Takeout...")
	// Forzamos el idioma inglés (hl=en) para que los selectores por aria-label funcionen siempre
	page := m.Browser.MustPage("https://takeout.google.com/settings/takeout?hl=en")
	page.MustWaitLoad()

	// Esperar a que el botón "Deselect all" esté visible y hacer clic
	fmt.Println("   - Deseleccionando todos los productos...")
	// Usamos selectores robustos basados en atributos que Google usa internamente
	page.MustElement(`[aria-label="Deselect all"]`).MustClick()
	time.Sleep(1 * time.Second) // Pequeña pausa para que la UI reaccione

	// Seleccionar solo Google Photos
	fmt.Println("   - Seleccionando Google Photos...")

	// Estrategia robusta: Buscar el texto "Google Photos" y subir por el DOM hasta encontrar el checkbox asociado
	// Esto evita depender de atributos data-id que pueden cambiar.
	productLabel := page.MustElementR("div", "Google Photos")

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
	fmt.Println("   - Avanzando al siguiente paso...")
	page.MustElement(`button[aria-label="Next step"]`).MustClick()

	// Esperar a que la sección de creación de exportación cargue
	page.MustWaitLoad()

	// Seleccionar 50GB para reducir número de archivos (menos ZIPs que descargar)
	fmt.Println("   - Configurando tamaño a 50GB...")
	// Abrir menú de tamaño
	page.MustElement(`div[aria-label="File size select"]`).MustClick()
	time.Sleep(500 * time.Millisecond)
	// Seleccionar opción de 50 GB
	page.MustElementR("li", "50 GB").MustClick()
	time.Sleep(500 * time.Millisecond)

	// Crear la exportación
	fmt.Println("   - Creando la exportación...")
	page.MustElementR("button", "Create export").MustClick()

	// Esperar a la página de confirmación
	fmt.Println("   - Esperando confirmación...")
	page.MustWaitNavigation()

	return nil
}
