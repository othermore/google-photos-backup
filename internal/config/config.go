package config

import (
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	UserID               string        `mapstructure:"user_id"`
	WorkingPath          string        `mapstructure:"working_path"`
	IndexPath            string        `mapstructure:"index_path"`
	ClientID             string        `mapstructure:"client_id"`
	ClientSecret         string        `mapstructure:"client_secret"`
	TokenPath            string        `mapstructure:"token_path"`
	BackupFrequency      time.Duration `mapstructure:"backup_frequency"`
	DownloadMode         string        `mapstructure:"download_mode"`          // "directDownload" or "driveDownload"
	FixAmbiguousMetadata string        `mapstructure:"fix_ambiguous_metadata"` // "yes", "no", "interactive"
	BackupPath           string        `mapstructure:"backup_path"`            // Where to store the final organized photos
}

const (
	ModeDirectDownload = "directDownload"
	ModeDriveDownload  = "driveDownload"
)

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
	viper.SetDefault("working_path", "./work")
	viper.SetDefault("index_path", "./index.jsonl")
	viper.SetDefault("backup_frequency", "168h") // 7 días (24*7)
	viper.SetDefault("download_mode", ModeDirectDownload)
	viper.SetDefault("fix_ambiguous_metadata", "interactive")
	viper.SetDefault("backup_path", "") // Empty by default

	// Definimos ruta por defecto para el token dentro del directorio de config
	if home, err := os.UserHomeDir(); err == nil {
		defaultTokenPath := filepath.Join(home, ".config", "google-photos-backup", "token.json")
		viper.SetDefault("token_path", defaultTokenPath)
	}

	// 4. Intentar leer
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Info(i18n.T("config_missing"))
		} else {
			logger.Error(i18n.T("config_read_error"), err)
			os.Exit(1)
		}
	}

	// 5. Cargar en estructura
	if err := viper.Unmarshal(&AppConfig); err != nil {
		logger.Error(i18n.T("config_decode_error"), err)
	}
}
