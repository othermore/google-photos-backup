package config

import (
	"fmt"
	"google-photos-backup/internal/i18n"
	"google-photos-backup/internal/logger"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	UserID               string   `mapstructure:"user_id"`
	WorkingPath          string   `mapstructure:"working_path"`
	EmailAlertTo         string   `mapstructure:"email_alert_to"`         // Destination email for alerts (uses system msmtp)
	FixAmbiguousMetadata string   `mapstructure:"fix_ambiguous_metadata"` // "yes", "no", "interactive"
	BackupPath           string   `mapstructure:"backup_path"`            // Where to store the final organized photos
	ImmichMasterEnabled  bool     `mapstructure:"immich_master_enabled"`  // Whether to maintain a master directory for Immich
	ImmichMasterPath     string   `mapstructure:"immich_master_path"`     // Relative path for Immich master directory
	RcloneRemote         string   `mapstructure:"rclone_remote"`          // Name of the rclone remote (default: "drive:")
	ValidMediaExtensions []string `mapstructure:"valid_media_extensions"` // List of valid media extensions
	IgnoredFiles         []string `mapstructure:"ignored_files"`          // List of items to ignore completely
}

const (
// Download modes removed as they are now command-specific
)

var AppConfig Config

func InitConfig(cfgDir string) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if cfgDir != "" {
		// Use specific config directory
		viper.AddConfigPath(cfgDir)
	} else {
		// Default search paths
		if runtime.GOOS == "linux" {
			viper.AddConfigPath("/etc/google-photos-backup/")
			viper.AddConfigPath("$HOME/.config/google-photos-backup")
		} else if runtime.GOOS == "darwin" { // macOS
			home, _ := os.UserHomeDir()
			viper.AddConfigPath(filepath.Join(home, ".config", "google-photos-backup"))
			viper.AddConfigPath(".") // Search in current folder too
		}
	}

	// 3. Default values
	viper.SetDefault("working_path", "./work")
	viper.SetDefault("fix_ambiguous_metadata", "interactive")
	viper.SetDefault("backup_path", "") // Empty by default
	viper.SetDefault("immich_master_enabled", false)
	viper.SetDefault("immich_master_path", "immich-master")
	viper.SetDefault("rclone_remote", "drive:")
	viper.SetDefault("email_alert_to", "")

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

	// 6. Inject defaults if not present
	defaultsInjected := false

	if len(AppConfig.ValidMediaExtensions) == 0 {
		AppConfig.ValidMediaExtensions = []string{
			"jpg", "nef", "orf", "mp4", "dng", "cr2", "cr3", "xmp", "jpeg", "png", "mov", "heic", "heif", "gif",
			"mpg", "mpeg", "mp", "3gp", "raw", "thm", "m4v", "psd", "ai", "wmv", "avi", "webp", "webm", "tif", "tiff", "mkv",
		}
		defaultsInjected = true
	}

	if len(AppConfig.IgnoredFiles) == 0 {
		AppConfig.IgnoredFiles = []string{
			"SYNOINDEX_MEDIA_INFO", "@eaDir", "*@synoeastream",
		}
		defaultsInjected = true
	}

	if defaultsInjected && viper.ConfigFileUsed() != "" {
		appendDefaultsToConfig(viper.ConfigFileUsed(), AppConfig.ValidMediaExtensions, AppConfig.IgnoredFiles)
	}
}

// appendDefaultsToConfig manually appends the default arrays to the end of the yaml file to preserve user's comments.
func appendDefaultsToConfig(configPath string, exts []string, ignores []string) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	text := string(content)

	needsAppend := false
	var builder strings.Builder

	if !strings.Contains(text, "valid_media_extensions:") {
		needsAppend = true
		builder.WriteString("\n# Lista de extensiones validas de medios a procesar\nvalid_media_extensions:\n")
		for _, e := range exts {
			builder.WriteString(fmt.Sprintf("  - \"%s\"\n", e))
		}
	}

	if !strings.Contains(text, "ignored_files:") {
		needsAppend = true
		builder.WriteString("\n# Lista de archivos y subdirectorios exactos o patrones que omitir por completo\nignored_files:\n")
		for _, idx := range ignores {
			builder.WriteString(fmt.Sprintf("  - \"%s\"\n", idx))
		}
	}

	if needsAppend {
		f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(builder.String())
			f.Close()
			logger.Info(i18n.T("config_appended_defaults"))
		}
	}
}

// IsValidMedia checks if the file has a configured valid extension
func IsValidMedia(filename string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	for _, validExt := range AppConfig.ValidMediaExtensions {
		if ext == strings.ToLower(validExt) {
			return true
		}
	}
	return false
}

// IsIgnored checks if the given file or directory base name should be completely ignored
func IsIgnored(path string) bool {
	base := filepath.Base(path)

	for _, pattern := range AppConfig.IgnoredFiles {
		// Exact Match
		if base == pattern {
			return true
		}
		// Glob Match
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}
	return false
}
