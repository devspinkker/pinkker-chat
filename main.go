package main

import (
	"PINKKER-CHAT/config"
	"PINKKER-CHAT/internal/chat/routes"
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	redisClient := setupRedisClient()
	defer redisClient.Close()

	newMongoDB := setupMongoDB()
	defer newMongoDB.Disconnect(context.Background())
	app := fiber.New()
	app.Use(cors.New())

	routes.Routes(app, redisClient, newMongoDB)

	PORT := config.PORT()
	if PORT == "" {
		PORT = "8080"
	}
	log.Fatal(app.Listen(":" + PORT))
}

func setupRedisClient() *redis.Client {
	PasswordRedis := config.PASSWORDREDIS()
	ADDRREDIS := config.ADDRREDIS()
	client := redis.NewClient(&redis.Options{
		Addr:     ADDRREDIS,
		Password: PasswordRedis,
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Error al conectar con Redis: %s", err.Error())
	}
	fmt.Println("Redis connect")
	return client
}
func setupMongoDB() *mongo.Client {
	URI := config.MONGODB_URI()
	if URI == "" {
		log.Fatal("MONGODB_URI FATAL")
	}

	clientOptions := options.Client().ApplyURI(URI)

	// Se crea el cliente de MongoDB
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal("MONGODB ERROR", err.Error())
	}

	// Se establece la conexión a la base de datos
	if err = client.Connect(context.Background()); err != nil {
		log.Fatal("MONGODB ERROR", err.Error())
	}

	// Se verifica que la conexión a la base de datos sea exitosa.
	if err = client.Ping(context.Background(), nil); err != nil {
		log.Fatal("MONGODB ERROR", err.Error())
	}
	return client
}
