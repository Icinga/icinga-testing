package utils

import "os"

// GetEnvDefault returns the value of an environment variable or a default value if the variable is not set.
func GetEnvDefault(key string, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
