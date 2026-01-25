package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

type Config struct {
	UserID       string `mapstructure:"user_id"`
	BackupPath   string `mapstructure:"backup_path"`
	IndexPath    string `mapstructure:"index_path"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	TokenPath    string `mapstructure:"token_path"`
}

var AppConfig Config

func InitConfig() {
	// 1. Definir nombre del fichero
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 2. Definir rutas de búsqueda según el Sistema Operativo
	if runtime.GOOS == "linux" {
		viper.AddConfigPath("/etc/google-photos-backup/")
		viper.AddConfigPath("$HOME/.config/google-photos-backup")
	} else if runtime.GOOS == "darwin" { // macOS
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(filepath.Join(home, ".config", "google-photos-backup"))
		viper.AddConfigPath(".") // Buscar en carpeta actual también
	}

	// 3. Valores por defecto
	viper.SetDefault("backup_path", "./backup")
	viper.SetDefault("index_path", "./index.jsonl")

	// Definimos ruta por defecto para el token dentro del directorio de config
	if home, err := os.UserHomeDir(); err == nil {
		defaultTokenPath := filepath.Join(home, ".config", "google-photos-backup", "token.json")
		viper.SetDefault("token_path", defaultTokenPath)
	}

	// 4. Intentar leer
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Aviso: No se encontró fichero config.yaml. Se usarán valores por defecto.")
		} else {
			fmt.Printf("Error leyendo config: %s\n", err)
			os.Exit(1)
		}
	}

	// 5. Cargar en estructura
	if err := viper.Unmarshal(&AppConfig); err != nil {
		fmt.Printf("Error decodificando config: %s\n", err)
	}
}
