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
