package routes

import (
	"PINKKER-CHAT/internal/chat/application"
	"PINKKER-CHAT/internal/chat/infrastructure"
	"PINKKER-CHAT/internal/chat/interfaces"
	"PINKKER-CHAT/pkg/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/redis/go-redis/v9"
)

func Routes(app *fiber.App, redisClient *redis.Client, MongoClient *mongo.Client) {

	Repository := infrastructure.NewRepository(redisClient, MongoClient)
	chatService := application.NewChatService(Repository)
	chatHandler := interfaces.NewChatHandler(chatService)

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return c.Status(fiber.StatusUpgradeRequired).SendString("Upgrade required")
	})

	// chat actions
	app.Post("/actionsChatStream", middleware.UseExtractor(), chatHandler.Actions)
	app.Post("/actionsModeratorChatStream", middleware.UseExtractor(), chatHandler.ActionModerator)

	// get commands
	app.Get("/getCommands", middleware.UseExtractor(), chatHandler.GetCommands)
	app.Post("/updataCommands", middleware.UseExtractor(), chatHandler.UpdataCommands)

	// chat messages
	app.Post("/chatStreaming/:roomID", middleware.UseExtractor(), chatHandler.SendMessage)
	var connectedUsers = make(map[string]bool)
	app.Get("/ws/chatStreaming/:roomID/:nameuser?", websocket.New(func(c *websocket.Conn) {
		fmt.Println("ASA")
		defer c.Close()
		roomID := c.Params("roomID")
		nameuser := c.Params("nameuser")
		if len(nameuser) >= 3 {
			roomIdObj, errinObjectID := primitive.ObjectIDFromHex(roomID)
			if errinObjectID != nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error con el id de la sala"))
				return
			}
			infoUser, err := Repository.GetUserInfo(roomIdObj, nameuser, false)
			if err != nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error con la info del usuario"))
				return

			}

			if infoUser.Baneado == true {
				c.WriteMessage(websocket.TextMessage, []byte("baneadoo"))
				return

			}
		}
		if !connectedUsers[nameuser] {
			connectedUsers[nameuser] = true
			UserConnectedStreamERR := chatHandler.UserConnectedStream(roomID, "connect")
			if UserConnectedStreamERR != nil {
				fmt.Println(UserConnectedStreamERR)
				return
			}
		}

		LastRoomMessages, err := chatHandler.RedisCacheGetLastRoomMessages(roomID)

		if err != nil {
			if err != redis.Nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error al unirse a la sala"))
				return
			}
		}
		for i := len(LastRoomMessages) - 1; i >= 0; i-- {
			err = c.WriteMessage(websocket.TextMessage, []byte(LastRoomMessages[i]))
			if err != nil {
				return
			}
		}

		for {
			errReceiveMessageFromRoom := chatHandler.ReceiveMessageFromRoom(c, connectedUsers)
			if errReceiveMessageFromRoom != nil {
				c.WriteMessage(websocket.TextMessage, []byte(errReceiveMessageFromRoom.Error()))
				return
			}
		}

	}))

}
