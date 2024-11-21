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

	return os.Getenv(signo)
}
