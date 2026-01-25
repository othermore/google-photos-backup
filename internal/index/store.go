package index

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// PhotoItem representa una línea en nuestro fichero de texto
type PhotoItem struct {
	ID           string    `json:"id"`
	GoogleURL    string    `json:"google_url"`
	Filename     string    `json:"filename"`
	IsBackedUp   bool      `json:"is_backed_up"`
	IsDeletedOnG bool      `json:"is_deleted_on_g"` // Borrada en Google
	LastCheck    time.Time `json:"last_check"`
}

type Store struct {
	FilePath string
	Photos   map[string]*PhotoItem // Mapa en memoria para acceso rápido
	mu       sync.RWMutex          // Para evitar problemas si escribimos y leemos a la vez
}

// NewStore crea una nueva instancia y carga el fichero si existe
func NewStore(path string) (*Store, error) {
	s := &Store{
		FilePath: path,
		Photos:   make(map[string]*PhotoItem),
	}
	// Si el fichero existe, lo cargamos (lo implementaremos en el siguiente paso)
	return s, nil
}

// Save (placeholder) Guardará todo el mapa al fichero plano
func (s *Store) Save() error {
	f, err := os.Create(s.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.Photos {
		if err := encoder.Encode(p); err != nil {
			return err
		}
	}
	return nil
}