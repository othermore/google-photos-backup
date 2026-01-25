package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"google-photos-backup/internal/utils"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Start inicia el flujo OAuth2: levanta servidor, abre navegador y guarda token
func Start(clientID, clientSecret, tokenPath string) error {
	// Configuración OAuth2
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/photoslibrary.readonly"},
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:54545/callback",
	}

	// Canal para saber cuándo termina el proceso
	done := make(chan error)

	// Servidor HTTP temporal para recibir el código de Google
	srv := &http.Server{Addr: ":54545"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No se recibió código de autorización", http.StatusBadRequest)
			return
		}

		// Intercambiar código por token
		token, err := config.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "Error al intercambiar token", http.StatusInternalServerError)
			done <- fmt.Errorf("exchange error: %w", err)
			return
		}

		// Guardar token en disco
		if err := saveToken(tokenPath, token); err != nil {
			http.Error(w, "Error guardando token", http.StatusInternalServerError)
			done <- fmt.Errorf("save token error: %w", err)
			return
		}

		fmt.Fprintf(w, "<h1>¡Autenticación Exitosa!</h1><p>Ya puedes cerrar esta ventana y volver a la terminal.</p>")
		done <- nil
	})

	// Iniciar servidor en segundo plano
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			done <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Abrir navegador
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Abriendo navegador... Si no se abre, visita: \n%s\n", authURL)
	utils.OpenBrowser(authURL)

	// Esperar a que termine el callback
	err := <-done
	srv.Shutdown(context.Background())
	return err
}

func saveToken(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory for token: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
