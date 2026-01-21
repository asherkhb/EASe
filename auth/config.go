// Package auth provides modular authentication backends and session management.
package auth

import (
	"flag"
	"os"
	"strconv"
)

// ConfigValue represents a configuration value that can come from a flag or environment variable.
// Flags take priority over environment variables.
type ConfigValue struct {
	FlagName    string
	EnvName     string
	Description string
}

// StringVar registers a string flag and returns a function to get the final value.
// The function checks flag first, then env var, then default.
func StringVar(fs *flag.FlagSet, flagName, envName, defaultVal, description string) *string {
	// Check env for default
	if envVal := os.Getenv(envName); envVal != "" {
		defaultVal = envVal
	}
	return fs.String(flagName, defaultVal, description+" (env: "+envName+")")
}

// BoolVar registers a bool flag and returns a function to get the final value.
func BoolVar(fs *flag.FlagSet, flagName, envName string, defaultVal bool, description string) *bool {
	// Check env for default
	if envVal := os.Getenv(envName); envVal != "" {
		if parsed, err := strconv.ParseBool(envVal); err == nil {
			defaultVal = parsed
		}
	}
	return fs.Bool(flagName, defaultVal, description+" (env: "+envName+")")
}

// IntVar registers an int flag and returns a function to get the final value.
func IntVar(fs *flag.FlagSet, flagName, envName string, defaultVal int, description string) *int {
	// Check env for default
	if envVal := os.Getenv(envName); envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil {
			defaultVal = parsed
		}
	}
	return fs.Int(flagName, defaultVal, description+" (env: "+envName+")")
}

// GetEnvOrDefault returns the environment variable value if set, otherwise the default.
func GetEnvOrDefault(envName, defaultVal string) string {
	if val := os.Getenv(envName); val != "" {
		return val
	}
	return defaultVal
}
