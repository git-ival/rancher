package environmentvariables

import (
	"os"
	"strconv"
)

func RancherCleanup() bool {
	rancherCleanupString := os.Getenv("RANCHER_CLEANUP")
	if rancherCleanupString == "false" || rancherCleanupString == "" {
		return false
	}

	return true
}

func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func ConvertStringToInt(inputString string) int {
	stringInt, err := strconv.Atoi(inputString)
	if err != nil {
		panic(err)
	}
	return stringInt
}
