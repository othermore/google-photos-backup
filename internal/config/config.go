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
	ImmichMasterEnabled  bool          `mapstructure:"immich_master_enabled"`  // Whether to maintain a master directory for Immich
	ImmichMasterPath     string        `mapstructure:"immich_master_path"`     // Relative path for Immich master directory
}

const (
	ModeDirectDownload = "directDownload"
	ModeDriveDownload  = "driveDownload"
)

var AppConfig Config

func InitConfig() {
	// 1. Define config filename
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 2. Define search paths based on OS
	if runtime.GOOS == "linux" {
		viper.AddConfigPath("/etc/google-photos-backup/")
		viper.AddConfigPath("$HOME/.config/google-photos-backup")
	} else if runtime.GOOS == "darwin" { // macOS
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(filepath.Join(home, ".config", "google-photos-backup"))
		viper.AddConfigPath(".") // Search in current folder too
	}

	// 3. Default values
	viper.SetDefault("working_path", "./work")
	viper.SetDefault("index_path", "./index.jsonl")
	viper.SetDefault("backup_frequency", "168h") // 7 days (24*7)
	viper.SetDefault("download_mode", ModeDirectDownload)
	viper.SetDefault("fix_ambiguous_metadata", "interactive")
	viper.SetDefault("fix_ambiguous_metadata", "interactive")
	viper.SetDefault("backup_path", "") // Empty by default
	viper.SetDefault("immich_master_enabled", false)
	viper.SetDefault("immich_master_path", "immich-master")

	// Define default path for token inside config directory
	if home, err := os.UserHomeDir(); err == nil {
		defaultTokenPath := filepath.Join(home, ".config", "google-photos-backup", "token.json")
		viper.SetDefault("token_path", defaultTokenPath)
	}

	// 4. Attempt to read
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Info(i18n.T("config_missing"))
		} else {
			logger.Error(i18n.T("config_read_error"), err)
			os.Exit(1)
		}
	}

	// 5. Load into struct
	if err := viper.Unmarshal(&AppConfig); err != nil {
		logger.Error(i18n.T("config_decode_error"), err)
	}
}
