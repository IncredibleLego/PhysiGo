package config

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Scale                   float64
	Fullscreen              bool
	InclinedObjectMode      string
	InclinedRotaryType      string
	InclinedRadius          float64
	InclinedMuR             float64
	InclinedMuRSet          bool
	InclinedTheta           float64
	InclinedMuS             float64
	InclinedMuK             float64
	InclinedMass            float64
	InclinedGravity         float64
	InclinedLength          float64
	InclinedHBlock          float64
	InclinedInitialVelocity float64
	InclinedMuSSet          bool
	InclinedMuKSet          bool
	InclinedGravitySet      bool
	InclinedWeightParallel  float64
	InclinedWeightPerp      float64
	InclinedNormal          float64
	InclinedStaticFriction  float64
	InclinedDynamicFriction float64
	InclinedNetForce        float64
	InclinedAcceleration    float64
	InclinedSlides          bool
	ProjectileV0            float64
	ProjectileTheta         float64
	ProjectileH             float64
	ProjectileRange         float64
	ProjectileTime          float64
	ProjectileGravity       float64
	ProjectileV0Set         bool
	ProjectileThetaSet      bool
	ProjectileHSet          bool
	ProjectileRangeSet      bool
	ProjectileTimeSet       bool
	ProjectileGravitySet    bool
	TextDimension           float64
	ScreenWidth             int
	ScreenHeight            int
	PopupWidth              int
	PopupHeight             int
	OptionsPerSecond        time.Duration
}

var GlobalConfig = &Config{
	Scale:                   1.0,
	Fullscreen:              true,
	InclinedObjectMode:      "block",
	InclinedRotaryType:      "",
	InclinedRadius:          0,
	InclinedMuR:             0,
	InclinedMuRSet:          false,
	InclinedTheta:           0,
	InclinedMuS:             0,
	InclinedMuK:             0,
	InclinedMass:            0,
	InclinedGravity:         9.8,
	InclinedLength:          0,
	InclinedHBlock:          0,
	InclinedInitialVelocity: 0,
	InclinedMuSSet:          false,
	InclinedMuKSet:          false,
	InclinedGravitySet:      false,
	InclinedWeightParallel:  0,
	InclinedWeightPerp:      0,
	InclinedNormal:          0,
	InclinedStaticFriction:  0,
	InclinedDynamicFriction: 0,
	InclinedNetForce:        0,
	InclinedAcceleration:    0,
	InclinedSlides:          false,
	ProjectileV0:            0,
	ProjectileTheta:         0,
	ProjectileH:             0,
	ProjectileRange:         0,
	ProjectileTime:          0,
	ProjectileGravity:       9.8,
	ProjectileV0Set:         false,
	ProjectileThetaSet:      false,
	ProjectileHSet:          false,
	ProjectileRangeSet:      false,
	ProjectileTimeSet:       false,
	ProjectileGravitySet:    false,
	TextDimension:           30,
	ScreenWidth:             960,
	ScreenHeight:            720,
	PopupWidth:              528, // 55% of height
	PopupHeight:             216, // 30% of height
	OptionsPerSecond:        time.Duration(time.Second / 4),
}

var DefaultConfig = &Config{
	Scale:                   1.0,
	Fullscreen:              true,
	InclinedObjectMode:      "block",
	InclinedRotaryType:      "",
	InclinedRadius:          0,
	InclinedMuR:             0,
	InclinedMuRSet:          false,
	InclinedTheta:           0,
	InclinedMuS:             0,
	InclinedMuK:             0,
	InclinedMass:            0,
	InclinedGravity:         9.8,
	InclinedLength:          0,
	InclinedHBlock:          0,
	InclinedInitialVelocity: 0,
	InclinedMuSSet:          false,
	InclinedMuKSet:          false,
	InclinedGravitySet:      false,
	InclinedWeightParallel:  0,
	InclinedWeightPerp:      0,
	InclinedNormal:          0,
	InclinedStaticFriction:  0,
	InclinedDynamicFriction: 0,
	InclinedNetForce:        0,
	InclinedAcceleration:    0,
	InclinedSlides:          false,
	ProjectileV0:            0,
	ProjectileTheta:         0,
	ProjectileH:             0,
	ProjectileRange:         0,
	ProjectileTime:          0,
	ProjectileGravity:       9.8,
	ProjectileV0Set:         false,
	ProjectileThetaSet:      false,
	ProjectileHSet:          false,
	ProjectileRangeSet:      false,
	ProjectileTimeSet:       false,
	ProjectileGravitySet:    false,
	TextDimension:           30,
	ScreenWidth:             960,
	ScreenHeight:            720,
	PopupWidth:              528, // 55% of height
	PopupHeight:             216, // 30% of height
	OptionsPerSecond:        time.Duration(time.Second / 4),
}

// Applica la scala ai valori di default e aggiorna la config
func ApplyScaleToConfig(cfg *Config, scale float64) {
	cfg.Scale = scale
	cfg.ScreenWidth = ((int(math.Round(float64(DefaultConfig.ScreenWidth)*scale)) + 5) / 10) * 10
	cfg.ScreenHeight = ((int(math.Round(float64(DefaultConfig.ScreenHeight)*scale)) + 5) / 10) * 10
	cfg.PopupWidth = int(math.Round(float64(DefaultConfig.PopupWidth) * scale))
	cfg.PopupHeight = int(math.Round(float64(DefaultConfig.PopupHeight) * scale))
	cfg.TextDimension = math.Round(DefaultConfig.TextDimension * scale)
}

// Cambia la scala e aggiorna la configurazione globale
func ChangeScale(newScale float64) error {
	ApplyScaleToConfig(GlobalConfig, newScale)
	return SaveConfig(GlobalConfig)
}

var configFilePath = resolveConfigFilePath()

func resolveConfigFilePath() string {
	execPath, err := os.Executable()
	if err != nil {
		return filepath.Join("config", "settings.json")
	}

	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, "config", "settings.json")
}

// SaveConfig saves the configuration to a JSON file.
func SaveConfig(config *Config) error {
	// Converts *config struct in JSON formatted with indentation ("" is prefix and "  " is indentation)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configFilePath), 0o755); err != nil {
		return err
	}
	// Create the file if it doesn't exist, or replace it if it does
	file, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer file.Close() // Close the file when the function returns
	// Write the JSON data to the file
	_, err = file.Write(data)
	// If all went well err is nil
	return err
}

// LoadConfig loads the configuration from a JSON file.
func LoadConfig(filePath string) (*Config, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close() // Close the file when the function returns
	// Create a new Config struct that will hold the loaded data
	var config Config
	// Decode the JSON data into the Config struct
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	// Returns a pointer to the loaded Config struct
	return &config, nil
}

// UpdateConfig updates the configuration using a provided function and saves it to the file.
func UpdateConfig(updateFunc func(*Config)) error {
	// Call the update function to modify the configuration
	updateFunc(GlobalConfig)
	// Save the updated configuration to the file
	return SaveConfig(GlobalConfig)
}

// InitConfig initializes the configuration by loading it from a file or using the default configuration.
func InitConfig() {
	// Check if the configuration file exists
	config, err := LoadConfig(configFilePath)
	// If the file doesn't exist or there's an error loading it, use the default configuration
	if err != nil {
		// Create the file with the default configuration
		_ = SaveConfig(DefaultConfig)
		GlobalConfig = DefaultConfig
	} else {
		GlobalConfig = config
	}
}
