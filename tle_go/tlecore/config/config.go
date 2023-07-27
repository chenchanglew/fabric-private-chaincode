package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func PrintConfig() {
	fmt.Println("--- viper config ---")
	settings := viper.AllSettings()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}
	fmt.Println("--finish viper config--")
}

func SetupConfig(configPath string) {
	viper.SetConfigFile(configPath)
	viper.ReadInConfig()
	viper.SetEnvPrefix("core")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}
