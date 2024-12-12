package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func PORT() string {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println(err.Error())

		log.Fatal("godotenv.Load error PORT")
	}
	return os.Getenv("PORT")
}

func TOKENPASSWORD() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error TOKENPASSWORD")
	}
	return os.Getenv("TOKENPASSWORD")
}

func PASSWORDREDIS() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error PASSWORDREDIS")
	}
	return os.Getenv("PASSWORDREDIS")
}

func ADDRREDIS() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error ADDRREDIS")
	}
	return os.Getenv("ADDRREDIS")
}
func VIP() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error VIP")
	}
	return os.Getenv("VIP")

}
func MODERATOR() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error MODERATOR")
	}
	return os.Getenv("MODERATOR")

}
func PARTNER() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error VERIFIED")
	}
	return os.Getenv("PARTNER")

}
func IDENTIDADMUTE() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error IDENTIDADMUTE")
	}
	return os.Getenv("IDENTIDADMUTE")

}
func MONGODB_URI() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error URI mongo")
	}
	return os.Getenv("MONGODB_URI")

}
func PINKKERPRIME() string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("godotenv.Load error URI mongo")
	}
	return os.Getenv("PINKKERPRIME")

}

func IdentidadSignoZodiacal(signo string) string {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error cargando el archivo .env")
	}

	// URL base para las imágenes
	baseURL := os.Getenv("BASE_ZODIACAL_URL")
	if baseURL == "" {
		baseURL = "https://www.pinkker.tv/uploads/assets/ZODIACAL"
	}

	signos := map[string]string{
		"ARIES":       "1.jpg",
		"TAURO":       "2.jpg",
		"GEMINIS":     "3.jpg",
		"CANCER":      "4.jpg",
		"LEO":         "5.jpg",
		"VIRGO":       "6.jpg",
		"LIBRA":       "7.jpg",
		"ESCORPIO":    "8.jpg",
		"SAGITARIO":   "9.jpg",
		"CAPRICORNIO": "10.jpg",
		"ACUARIO":     "11.jpg",
		"PISCIS":      "12.jpg",
	}

	fileName, exists := signos[signo]
	if !exists {
		return fmt.Sprintf("Signo zodiacal '%s' no encontrado", signo)
	}

	return fmt.Sprintf("%s/%s", baseURL, fileName)
}
