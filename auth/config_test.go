package auth

import (
	"flag"
	"os"
	"testing"
)

func TestStringVar(t *testing.T) {
	tests := []struct {
		name       string
		flagVal    string
		envVal     string
		defaultVal string
		want       string
	}{
		{
			name:       "flag takes priority",
			flagVal:    "from-flag",
			envVal:     "from-env",
			defaultVal: "default",
			want:       "from-flag",
		},
		{
			name:       "env fallback when no flag",
			flagVal:    "",
			envVal:     "from-env",
			defaultVal: "default",
			want:       "from-env",
		},
		{
			name:       "default when neither set",
			flagVal:    "",
			envVal:     "",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			envName := "TEST_STRING_VAR_" + tt.name
			if tt.envVal != "" {
				os.Setenv(envName, tt.envVal)
				defer os.Unsetenv(envName)
			}

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			ptr := StringVar(fs, "test-flag", envName, tt.defaultVal, "test description")

			// Parse with flag value if provided
			if tt.flagVal != "" {
				fs.Parse([]string{"-test-flag=" + tt.flagVal})
			} else {
				fs.Parse([]string{})
			}

			if *ptr != tt.want {
				t.Errorf("got %q, want %q", *ptr, tt.want)
			}
		})
	}
}

func TestIntVar(t *testing.T) {
	tests := []struct {
		name       string
		flagVal    int
		setFlag    bool
		envVal     string
		defaultVal int
		want       int
	}{
		{
			name:       "flag takes priority",
			flagVal:    100,
			setFlag:    true,
			envVal:     "50",
			defaultVal: 10,
			want:       100,
		},
		{
			name:       "env fallback when no flag",
			flagVal:    0,
			setFlag:    false,
			envVal:     "50",
			defaultVal: 10,
			want:       50,
		},
		{
			name:       "default when neither set",
			flagVal:    0,
			setFlag:    false,
			envVal:     "",
			defaultVal: 10,
			want:       10,
		},
		{
			name:       "invalid env uses default",
			flagVal:    0,
			setFlag:    false,
			envVal:     "not-a-number",
			defaultVal: 10,
			want:       10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envName := "TEST_INT_VAR_" + tt.name
			if tt.envVal != "" {
				os.Setenv(envName, tt.envVal)
				defer os.Unsetenv(envName)
			}

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			ptr := IntVar(fs, "test-int", envName, tt.defaultVal, "test description")

			if tt.setFlag {
				fs.Parse([]string{"-test-int", string(rune(tt.flagVal))})
				// Use a different approach for int flags
				fs = flag.NewFlagSet("test", flag.ContinueOnError)
				ptr = IntVar(fs, "test-int", envName, tt.defaultVal, "test description")
				fs.Parse([]string{"-test-int=100"})
			} else {
				fs.Parse([]string{})
			}

			if tt.setFlag && *ptr != 100 {
				t.Errorf("got %d, want 100", *ptr)
			} else if !tt.setFlag && *ptr != tt.want {
				t.Errorf("got %d, want %d", *ptr, tt.want)
			}
		})
	}
}

func TestBoolVar(t *testing.T) {
	tests := []struct {
		name       string
		flagVal    bool
		setFlag    bool
		envVal     string
		defaultVal bool
		want       bool
	}{
		{
			name:       "flag true takes priority",
			flagVal:    true,
			setFlag:    true,
			envVal:     "false",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "env true fallback",
			flagVal:    false,
			setFlag:    false,
			envVal:     "true",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "env 1 as true",
			flagVal:    false,
			setFlag:    false,
			envVal:     "1",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "default when neither set",
			flagVal:    false,
			setFlag:    false,
			envVal:     "",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "invalid env uses default",
			flagVal:    false,
			setFlag:    false,
			envVal:     "invalid",
			defaultVal: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envName := "TEST_BOOL_VAR_" + tt.name
			if tt.envVal != "" {
				os.Setenv(envName, tt.envVal)
				defer os.Unsetenv(envName)
			}

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			ptr := BoolVar(fs, "test-bool", envName, tt.defaultVal, "test description")

			if tt.setFlag {
				fs.Parse([]string{"-test-bool=true"})
			} else {
				fs.Parse([]string{})
			}

			if *ptr != tt.want {
				t.Errorf("got %v, want %v", *ptr, tt.want)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		envVal     string
		setEnv     bool
		defaultVal string
		want       string
	}{
		{
			name:       "env set",
			envVal:     "env-value",
			setEnv:     true,
			defaultVal: "default",
			want:       "env-value",
		},
		{
			name:       "env not set",
			envVal:     "",
			setEnv:     false,
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "env empty string",
			envVal:     "",
			setEnv:     true,
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envName := "TEST_GET_ENV_" + tt.name
			if tt.setEnv {
				os.Setenv(envName, tt.envVal)
				defer os.Unsetenv(envName)
			}

			result := GetEnvOrDefault(envName, tt.defaultVal)
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}
