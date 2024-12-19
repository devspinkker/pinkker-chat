package routes

import (
	"PINKKER-CHAT/internal/chat/application"
	"PINKKER-CHAT/internal/chat/infrastructure"
	"PINKKER-CHAT/internal/chat/interfaces"
	"PINKKER-CHAT/pkg/jwt"
	"PINKKER-CHAT/pkg/middleware"
	"encoding/json"
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

	app.Post("/GetInfoUserInRoom", middleware.UseExtractor(), chatHandler.GetInfoUserInRoom)
	app.Get("/GetMessagesForSecond", chatHandler.GetMessagesForSecond)
	// chat actions
	app.Post("/actionsChatStream", middleware.UseExtractor(), chatHandler.Actions)
	app.Post("/actionsModeratorChatStream", middleware.UseExtractor(), chatHandler.ActionModerator)
	app.Post("/ActionIdentidadUser", middleware.UseExtractor(), chatHandler.ActionIdentidadUser)
	app.Post("/RedisFindMatchingUsersInRoomByPrefix", middleware.UseExtractor(), chatHandler.RedisFindMatchingUsersInRoomByPrefix)
	// get commands
	app.Get("/getCommands", middleware.UseExtractor(), chatHandler.GetCommands)
	app.Post("/updataCommands", middleware.UseExtractor(), chatHandler.UpdataCommands)

	// chat messages
	app.Post("/chatStreaming/:roomID", middleware.UseExtractor(), chatHandler.SendMessage)
	app.Get("/ws/chatStreaming/:roomID/:token", websocket.New(func(c *websocket.Conn) {
		roomID := c.Params("roomID")
		token := c.Params("token", "null")
		var nameuser string
		var id primitive.ObjectID

		var verified bool
		roomIdObj, errinObjectID := primitive.ObjectIDFromHex(roomID)
		if errinObjectID != nil {
			c.WriteMessage(websocket.TextMessage, []byte("Error con el id de la sala"))
			return
		}
		if token != "null" {
			nameuserExtractDataFromToken, idUser, partner, err := jwt.ExtractDataFromToken(token)
			if err != nil {
				return
			}
			id, errinObjectID = primitive.ObjectIDFromHex(idUser)
			if errinObjectID != nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error con el id de la sala"))
				return
			}
			nameuser = nameuserExtractDataFromToken
			verified = partner

		}

		if len(nameuser) >= 4 {
			infoUser, err := chatHandler.InfoUserRoomChache(roomIdObj, nameuser, verified)
			if err != nil {
				c.WriteMessage(websocket.TextMessage, []byte("Error con la info del usuario"))
				return

			}

			if infoUser.Baneado {
				c.WriteMessage(websocket.TextMessage, []byte("baneado"))
			}
		}
		if len(nameuser) >= 4 {
			UserConnectedStreamERR := chatHandler.UserConnectedStream(roomID, nameuser, "connect", id)
			if UserConnectedStreamERR != nil {
				c.Close()
				return
			}
		}
		LastRoomMessages, err := chatHandler.RedisCacheGetLastRoomMessages(roomID)

		if err != nil {
			if err != redis.Nil {
				chatHandler.UserConnectedStream(roomID, nameuser, "disconnect", id)
				c.WriteMessage(websocket.TextMessage, []byte("Error al unirse a la sala"))
				c.Close()
				return
			}
		}
		for i := len(LastRoomMessages) - 1; i >= 0; i-- {
			err = c.WriteMessage(websocket.TextMessage, []byte(LastRoomMessages[i]))
			if err != nil {
				chatHandler.UserConnectedStream(roomID, nameuser, "disconnect", id)
				c.Close()
				return
			}
		}
		for {

			errReceiveMessageFromRoom := chatHandler.ReceiveMessageFromRoom(c)
			if errReceiveMessageFromRoom != nil {
				c.WriteMessage(websocket.TextMessage, []byte(errReceiveMessageFromRoom.Error()))
				chatHandler.UserConnectedStream(roomID, nameuser, "disconnect", id)
				c.Close()
				return
			}
		}

	}))

	// Agregar un nuevo punto final para eliminar mensajes
	app.Delete("/chatStreaming/:roomID/messages/delete/:messageID", middleware.UseExtractor(), chatHandler.DeleteMessage)
	app.Post("/chatStreaming/:roomID/messages/anclar", middleware.UseExtractor(), chatHandler.AnclarMessage)
	app.Get("/chatStreaming/:roomID/messages/desanclar/:messageID", middleware.UseExtractor(), chatHandler.DesanclarMessage)
	app.Post("/chatStreaming/:roomID/messages/Host", middleware.UseExtractor(), chatHandler.Host)

	app.Get("/ws/notifications/notifications/actionMessages/:roomID", websocket.New(func(c *websocket.Conn) {
		roomID := c.Params("roomID") + "actionMessages"

		// Validar el WebSocket antes de proceder
		if c == nil {
			fmt.Println("Error: WebSocket connection is nil.")
			return
		}

		// Recuperar mensaje anclado si existe
		ancladoMessage, err := chatHandler.GetAncladoMessageFromRedis(roomID)
		if err == nil && ancladoMessage != nil {
			notification := map[string]interface{}{
				"action":  "message_Anclar",
				"message": ancladoMessage,
			}

			// Intentar enviar el mensaje anclado
			jsonData, err := json.Marshal(notification)
			if err == nil {
				err = c.WriteMessage(websocket.TextMessage, jsonData)
				if err != nil {
					fmt.Println("Error writing anchored message:", err)
					c.Close()
					return
				}
			}
		}

		// Manejar mensajes desde el WebSocket
		errReceiveMessageFromRoom := chatHandler.ReceiveMessageActionMessages(c)
		if errReceiveMessageFromRoom != nil {
			fmt.Println("Error receiving messages:", errReceiveMessageFromRoom)
			_ = c.WriteMessage(websocket.TextMessage, []byte(errReceiveMessageFromRoom.Error()))
			c.Close()
			return
		}
	}))

}
