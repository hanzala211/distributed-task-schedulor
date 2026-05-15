package env

import (
	"os"
	"strconv"
)

func GetEnv(key string, fallback string) string {
	str := os.Getenv(key)
	if str == "" {
		return fallback
	}
	return str
}

func GetEnvInt(key string, fallback int) int {
	str := os.Getenv(key)
	if str == "" {
		return fallback
	}
	num, err := strconv.Atoi(str)
	if err != nil {
		return fallback
	}
	return num
}

func GetEnvBool(key string, fallback bool) bool {
	str := os.Getenv(key)
	if str == "" {
		return fallback
	}
	boolean, err := strconv.ParseBool(str)
	if err != nil {
		return fallback
	}
	return boolean
}
