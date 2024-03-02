package routes

import (
	"PINKKER-CHAT/internal/chat/application"
	"PINKKER-CHAT/internal/chat/infrastructure"
	"PINKKER-CHAT/internal/chat/interfaces"
	"PINKKER-CHAT/pkg/jwt"
	"PINKKER-CHAT/pkg/middleware"
	"PINKKER-CHAT/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/redis/go-redis/v9"
)

func Routes(app *fiber.App, redisClient *redis.Client, MongoClient *mongo.Client) {

	Repository := infrastructure.NewRepository(redisClient, MongoClient)
	chatService := application.NewChatService(Repository)
	chatHandler := interfaces.NewChatHandler(chatService)
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOrigins:     "https://www.pinkker.tv",
		AllowHeaders:     "Origin, Content-Type, Accept, Accept-Language, Content-Length",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return c.Status(fiber.StatusUpgradeRequired).SendString("Upgrade required")
	})

	app.Post("/GetInfoUserInRoom", middleware.UseExtractor(), chatHandler.GetInfoUserInRoom)
	// chat actions
	app.Post("/actionsChatStream", middleware.UseExtractor(), chatHandler.Actions)
	app.Post("/actionsModeratorChatStream", middleware.UseExtractor(), chatHandler.ActionModerator)

	// get commands
	app.Get("/getCommands", middleware.UseExtractor(), chatHandler.GetCommands)
	app.Post("/updataCommands", middleware.UseExtractor(), chatHandler.UpdataCommands)

	// chat messages
	app.Post("/chatStreaming/:roomID", middleware.UseExtractor(), chatHandler.SendMessage)
	var connectedUsers = utils.NewConnectedUsers()
	app.Get("/ws/chatStreaming/:roomID/:token", websocket.New(func(c *websocket.Conn) {
		roomID := c.Params("roomID")
		token := c.Params("token", "null")
		var nameuser string
		var verified bool
		roomIdObj, errinObjectID := primitive.ObjectIDFromHex(roomID)
		if errinObjectID != nil {
			c.WriteMessage(websocket.TextMessage, []byte("Error con el id de la sala"))
			return
		}
		if token != "null" {
			nameuserExtractDataFromToken, _, verifiedToken, err := jwt.ExtractDataFromToken(token)
			if err != nil {
				return
			}
			nameuser = nameuserExtractDataFromToken
			verified = verifiedToken

		}

		if len(nameuser) >= 3 {
			infoUser, err := chatHandler.InfoUserRoomChache(roomIdObj, nameuser, verified)
			if err != nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error con la info del usuario"))
				return

			}

			if infoUser.Baneado {
				c.WriteMessage(websocket.TextMessage, []byte("baneado"))
			}
		}
		if !connectedUsers.Get(nameuser) && len(nameuser) >= 4 {
			connectedUsers.Set(nameuser, true)
			UserConnectedStreamERR := chatHandler.UserConnectedStream(roomID, "connect")
			if UserConnectedStreamERR != nil {
				c.Close()
				return
			}
		}
		LastRoomMessages, err := chatHandler.RedisCacheGetLastRoomMessages(roomID)

		if err != nil {
			if err != redis.Nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error al unirse a la sala"))
				c.Close()
				return
			}
		}
		for i := len(LastRoomMessages) - 1; i >= 0; i-- {
			err = c.WriteMessage(websocket.TextMessage, []byte(LastRoomMessages[i]))
			if err != nil {
				c.Close()

				return
			}
		}
		for {
			errReceiveMessageFromRoom := chatHandler.ReceiveMessageFromRoom(c, nameuser, connectedUsers)
			if errReceiveMessageFromRoom != nil {
				c.Close()
				c.WriteMessage(websocket.TextMessage, []byte(errReceiveMessageFromRoom.Error()))
				return
			}
		}

	}))

}
