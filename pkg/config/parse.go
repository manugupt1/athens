package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/gomods/athens/pkg/errors"
	"github.com/kelseyhightower/envconfig"
	validator "gopkg.in/go-playground/validator.v9"
)

// Config provides configuration values for all components
type Config struct {
	TimeoutConf
	GoEnv                string `validate:"required" envconfig:"GO_ENV"`
	GoBinary             string `validate:"required" envconfig:"GO_BINARY_PATH"`
	GoGetWorkers         int    `validate:"required" envconfig:"ATHENS_GOGET_WORKERS"`
	ProtocolWorkers      int    `validate:"required" envconfig:"ATHENS_PROTOCOL_WORKERS"`
	LogLevel             string `validate:"required" envconfig:"ATHENS_LOG_LEVEL"`
	BuffaloLogLevel      string `validate:"required" envconfig:"BUFFALO_LOG_LEVEL"`
	MaxConcurrency       int    `envconfig:"ATHENS_MAX_CONCURRENCY"`  // only used by Olympus. TODO: remove.
	MaxWorkerFails       uint   `envconfig:"ATHENS_MAX_WORKER_FAILS"` // only used by Olympus. TODO: remove.
	CloudRuntime         string `validate:"required" envconfig:"ATHENS_CLOUD_RUNTIME"`
	FilterFile           string `envconfig:"ATHENS_FILTER_FILE"`
	EnableCSRFProtection bool   `envconfig:"ATHENS_ENABLE_CSRF_PROTECTION"`
	TraceExporterURL     string `envconfig:"ATHENS_TRACE_EXPORTER_URL"`
	TraceExporter        string `envconfig:"ATHENS_TRACE_EXPORTER"`
	Proxy                *ProxyConfig
	Olympus              *OlympusConfig `validate:"-"` // ignoring validation until Olympus is up.
	Storage              *StorageConfig
}

// ParseConfigFile parses the given file into an athens config struct
func ParseConfigFile(configFile string) (*Config, error) {

	var config Config
	// attempt to read the given config file
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, err
	}

	// Check file perms from config
	if err := checkFilePerms(config.FilterFile); err != nil {
		return nil, err
	}

	// override values with environment variables if specified
	if err := envOverride(&config); err != nil {
		return nil, err
	}

	// set default values
	setRuntimeDefaults(&config)

	// If not defined, set storage timeouts to global timeout
	setStorageTimeouts(config.Storage, config.Timeout)

	// delete invalid storage backend configs
	// envconfig initializes *all* struct pointers, even if there are no corresponding defaults or env variables
	// this method prunes all such invalid configurations
	deleteInvalidStorageConfigs(config.Storage)

	// validate all required fields have been populated
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return &config, nil
}

func setRuntimeDefaults(config *Config) {
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = runtime.NumCPU()
	}
}

// envOverride uses Environment variables to override unspecified properties
func envOverride(config *Config) error {
	if err := envconfig.Process("athens", config); err != nil {
		return err
	}
	return nil
}

func validateConfig(c Config) error {
	validate := validator.New()
	err := validate.Struct(c)
	if err != nil {
		return err
	}
	return nil
}

// GetConf accepts the path to a file, constructs an absolute path to the file,
// and attempts to parse it into a Config struct.
func GetConf(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to construct absolute path to test config file")
	}
	conf, err := ParseConfigFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse test config file: %s", err.Error())
	}
	return conf, nil
}

// checkFilePerms given a list of files
func checkFilePerms(files ...string) error {
	const op = "config.checkFilePerms"

	for _, f := range files {
		if f == "" {
			continue
		}

		// TODO: Remove opening a file
		// There is a subtle bug in the filter module which ignores the filter file if it does not find it.
		// This check can be removed once that has been fixed
		_, err := os.Open(f)
		if err != nil {
			continue
		}

		fInfo, err := os.Lstat(f)
		if err != nil {
			return errors.E(op, err)
		}

		if fInfo.Mode() != 0600 {
			return errors.E(op, f+" should have 0660 as permission")
		}
	}

	return nil
}
