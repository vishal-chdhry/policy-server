package utils

import "os"

func LookupEnvOrDefault(env_name, fallback string) string {
	if value, ok := os.LookupEnv(env_name); ok {
		return value
	} else {
		return fallback
	}
}
