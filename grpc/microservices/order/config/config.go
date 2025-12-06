package config
import (
	"log"
	"os"
	"strconv"
)

func GetEnv() string {
	return getEnvironmentValue("ENV")
}

func GetDataSourceURL() string {
	return getEnvironmentValue("DATA_SOURCE_URL")
}

func GetApplicationPort() int {
	portStr := getEnvironmentValue("APPLICATION_PORT")

	// Atoi = ASCII to Integer. Converte texto "8080" para número 8080
	port, err := strconv.Atoi(portStr)

	if err != nil {
		log.Fatalf("port: %s is invalid", portStr)
	}

	return port
}

func getEnvironmentValue(key string) string {
	// Tenta pegar o valor da chave (ex: "DATA_SOURCE_URL")
	if os.Getenv(key) == "" {
		// SE o valor estiver vazio, o programa CRASHA propositalmente.
		log.Fatalf("%s environment variable is missing", key)
	}
	return os.Getenv(key)
}

