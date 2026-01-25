package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

type Config struct {
	UserID     string `mapstructure:"user_id"`
	BackupPath string `mapstructure:"backup_path"`
	IndexPath  string `mapstructure:"index_path"`
    // Aquí añadiremos credenciales más adelante
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
