package api

import (
	"context"
	"encoding/json"
	"fmt"
	"google-photos-backup/internal/config"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	apiBaseURL = "https://photoslibrary.googleapis.com/v1"
)

// Client maneja la comunicación con la API de Google Photos
type Client struct {
	http *http.Client
}

// MediaItem representa una foto o video en la respuesta de la API
type MediaItem struct {
	ID            string        `json:"id"`
	ProductURL    string        `json:"productUrl"`
	BaseURL       string        `json:"baseUrl"`
	MimeType      string        `json:"mimeType"`
	Filename      string        `json:"filename"`
	MediaMetadata MediaMetadata `json:"mediaMetadata"`
}

type MediaMetadata struct {
	CreationTime time.Time `json:"creationTime"`
	Width        string    `json:"width"`
	Height       string    `json:"height"`
}

type listResponse struct {
	MediaItems    []MediaItem `json:"mediaItems"`
	NextPageToken string      `json:"nextPageToken"`
}

// NewClient crea un cliente HTTP autenticado usando el token guardado
func NewClient() (*Client, error) {
	// 1. Cargar el token desde el archivo
	tokenFile, err := os.Open(config.AppConfig.TokenPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir token.json (ejecuta 'configure' primero): %w", err)
	}
	defer tokenFile.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(tokenFile).Decode(tok); err != nil {
		return nil, fmt.Errorf("token.json corrupto: %w", err)
	}

	// 2. Configurar OAuth2 (necesario para el refresco automático)
	conf := &oauth2.Config{
		ClientID:     config.AppConfig.ClientID,
		ClientSecret: config.AppConfig.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/photoslibrary.readonly"},
		Endpoint:     google.Endpoint,
	}

	// 3. Crear el cliente
	ctx := context.Background()
	client := conf.Client(ctx, tok)

	return &Client{http: client}, nil
}

// ListMediaItems lista fotos de la biblioteca con paginación
func (c *Client) ListMediaItems(pageSize int, pageToken string) ([]MediaItem, string, error) {
	url := fmt.Sprintf("%s/mediaItems?pageSize=%d", apiBaseURL, pageSize)
	if pageToken != "" {
		url += "&pageToken=" + pageToken
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("error en petición API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("la API retornó estado: %s", resp.Status)
	}

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("error decodificando respuesta JSON: %w", err)
	}

	return result.MediaItems, result.NextPageToken, nil
}
