package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// processEnv gives process variables priority over a development .env file.
// Production never reads .env, even when the file exists.
func processEnv(appEnv string) lookupFunc {
	fileValues := map[string]string{}
	if appEnv != "production" {
		fileValues = readDotEnv()
	}
	return func(key string) (string, bool) {
		if value, ok := os.LookupEnv(key); ok {
			return value, true
		}
		value, ok := fileValues[key]
		return value, ok
	}
}

func readDotEnv() map[string]string {
	if values, err := godotenv.Read(); err == nil {
		return values
	}
	executable, err := os.Executable()
	if err != nil {
		return map[string]string{}
	}
	values, err := godotenv.Read(filepath.Join(filepath.Dir(executable), ".env"))
	if err != nil {
		return map[string]string{}
	}
	return values
}
