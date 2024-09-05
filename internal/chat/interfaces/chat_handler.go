package interfaces

import (
	"PINKKER-CHAT/internal/chat/application"
	"PINKKER-CHAT/internal/chat/domain"
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatHandler struct {
	chatService *application.ChatService
}

func NewChatHandler(chatService *application.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}
func (h *ChatHandler) RedisFindMatchingUsersInRoomByPrefix(c *fiber.Ctx) error {
	var req domain.RedisFindActiveUserInRoomByNamePrefix
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}
	usersActives, err := h.chatService.RedisFindMatchingUsersInRoomByPrefix(req.Room, req.NameUser)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "StatusBadRequest",
			"active":  usersActives,
			"data":    err,
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "ok",
		"active":  usersActives,
	})

}
func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	NameUser := c.Context().UserValue("nameUser").(string)
	verified := c.Context().UserValue("verified").(bool)
	roomID := c.Params("roomID")
	var req domain.MessagesTheSendMessagesRoom
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}

	if err := req.MessagesTheSendMessagesRoomValidate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": err.Error(),
		})
	}
	room, errinObjectID := primitive.ObjectIDFromHex(roomID)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	errPublishMessageInRoom := h.chatService.PublishMessageInRoom(room, req.Message, req.ResNameUser, req.ResMessage, NameUser, verified)
	if errPublishMessageInRoom != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": errPublishMessageInRoom.Error(),
		})
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "ok",
		"data":    "send message",
	})

}
func (h *ChatHandler) DeleteMessage(c *fiber.Ctx) error {
	roomID := c.Params("roomID")
	messageID := c.Params("messageID")
	verified := c.Context().UserValue("verified").(bool)
	NameUser := c.Context().UserValue("nameUser").(string)

	IdUserTokenP, errinObjectID := primitive.ObjectIDFromHex(roomID)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	infoUserAuth, errGetUserInfo := h.chatService.GetUserInfoStruct(IdUserTokenP, NameUser, verified)

	if errGetUserInfo != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": "StatusInternalServerError",
		})
	}
	if !infoUserAuth.Moderator && !infoUserAuth.StreamerChannelOwner {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "action not authorized",
		})
	}

	// // Eliminar el mensaje
	// err := h.chatService.DeleteMessage(roomID, messageID)
	// if err != nil {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"error": "Error deleting message",
	// 	})
	// }

	// Envía una notificación al frontend indicando que el mensaje fue eliminado
	err := h.NotifyMessageDeletedToRoomClients(roomID, messageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error notifying message deletion",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Message deleted successfully",
	})
}
func (h *ChatHandler) AnclarMessage(c *fiber.Ctx) error {
	roomID := c.Params("roomID")
	var data domain.AnclarMessageData

	// Decodificar los datos del cuerpo de la solicitud
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request data",
		})
	}

	// Extraer los datos del usuario del contexto
	verified := c.Context().UserValue("verified").(bool)
	NameUser := c.Context().UserValue("nameUser").(string)

	// Validar si el usuario tiene permisos
	IdUserTokenP, errinObjectID := primitive.ObjectIDFromHex(roomID)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	infoUserAuth, errGetUserInfo := h.chatService.GetUserInfoStruct(IdUserTokenP, NameUser, verified)
	if errGetUserInfo != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": "StatusInternalServerError",
		})
	}
	if !infoUserAuth.Moderator && !infoUserAuth.StreamerChannelOwner {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "action not authorized",
		})
	}

	err := h.NotifyMessageAnclarToRoomClients(roomID, data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error notifying message anchoring",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Message anchored successfully",
	})
}

func (h *ChatHandler) DesanclarMessage(c *fiber.Ctx) error {
	roomID := c.Params("roomID")
	messageID := c.Params("messageID")
	verified := c.Context().UserValue("verified").(bool)
	NameUser := c.Context().UserValue("nameUser").(string)

	IdUserTokenP, errinObjectID := primitive.ObjectIDFromHex(roomID)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	infoUserAuth, errGetUserInfo := h.chatService.GetUserInfoStruct(IdUserTokenP, NameUser, verified)

	if errGetUserInfo != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": "StatusInternalServerError",
		})
	}
	if !infoUserAuth.Moderator && !infoUserAuth.StreamerChannelOwner {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "action not authorized",
		})
	}

	err := h.NotifyMessageDesanclarToRoomClients(roomID, messageID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error notifying message deletion",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Message deleted successfully",
	})
}

func (h *ChatHandler) Host(c *fiber.Ctx) error {
	roomID := c.Params("roomID")
	var data domain.Host

	// Decodificar los datos del cuerpo de la solicitud
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request data",
		})
	}

	// Extraer los datos del usuario del contexto
	verified := c.Context().UserValue("verified").(bool)
	NameUser := c.Context().UserValue("nameUser").(string)

	// Validar si el usuario tiene permisos
	IdUserTokenP, errinObjectID := primitive.ObjectIDFromHex(roomID)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	infoUserAuth, errGetUserInfo := h.chatService.GetUserInfoStruct(IdUserTokenP, NameUser, verified)
	if errGetUserInfo != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": "StatusInternalServerError",
		})
	}
	if !infoUserAuth.StreamerChannelOwner {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "action not authorized",
		})
	}

	err := h.NotifyHost(roomID, NameUser, data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error notifying message anchoring",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Message anchored successfully",
	})
}
func (h *ChatHandler) NotifyHost(roomID string, byHost string, Host domain.Host) error {

	notification := map[string]interface{}{
		"action": "host_action",
		"hostA":  Host,
	}
	h.chatService.PublishAction(roomID+"action", notification)

	stream, err := h.chatService.FindStreamByStreamer(Host.NameUser)
	if err != nil {
		return err
	}

	notification = map[string]interface{}{
		"action":     "Host",
		"hostby":     byHost,
		"spectators": stream.ViewerCount,
	}
	h.chatService.PublishAction(stream.ID.Hex()+"action", notification)

	return nil
}

func (h *ChatHandler) NotifyMessageAnclarToRoomClients(roomID string, anclarMessage domain.AnclarMessageData) error {

	notification := map[string]interface{}{
		"action":  "message_Anclar",
		"message": anclarMessage,
	}
	err := h.chatService.SaveMessageAnclarRedis(roomID+"action", anclarMessage)
	if err != nil {
		return err
	}
	h.chatService.PublishAction(roomID+"action", notification)

	return nil

}

func (h *ChatHandler) GetAncladoMessageFromRedis(roomID string) (map[string]interface{}, error) {
	return h.chatService.GetAncladoMessageFromRedis(roomID + "action")
}
func (h *ChatHandler) NotifyMessageDesanclarToRoomClients(roomID, messageID string) error {

	notification := map[string]interface{}{
		"action":     "message_Desanclar",
		"message_id": messageID,
	}

	return h.chatService.PublishAction(roomID+"action", notification)
}
func (h *ChatHandler) NotifyMessageDeletedToRoomClients(roomID, messageID string) error {

	notification := map[string]interface{}{
		"action":     "message_deleted",
		"message_id": messageID,
	}

	return h.chatService.PublishAction(roomID+"action", notification)

}

func (h *ChatHandler) UserConnectedStream(roomID, nameUser, action string) error {

	err := h.chatService.UserConnectedStream(roomID, nameUser, action)

	return err
}

func (h *ChatHandler) InfoUserRoomChache(roomID primitive.ObjectID, nameUser string, verified bool) (domain.UserInfo, error) {

	UserInfo, err := h.chatService.InfoUserRoomChache(roomID, nameUser, verified)

	return UserInfo, err
}

func (h *ChatHandler) ReceiveMessageFromRoom(c *websocket.Conn) error {
	roomID := c.Params("roomID")

	sub := h.chatService.SubscribeToRoom(roomID)

	for {
		go func() {
			for {
				_, _, err := c.ReadMessage()
				if err != nil {
					h.chatService.CloseSubscription(sub)
					c.Close()
					return
				}
			}
		}()

		message, err := sub.ReceiveMessage(context.Background())
		if err != nil {
			h.chatService.CloseSubscription(sub)
			return err
		}

		err = c.WriteMessage(websocket.TextMessage, []byte(message.Payload))
		if err != nil {
			h.chatService.CloseSubscription(sub)
			return err
		}
	}
}
func (h *ChatHandler) ReceiveMessageActionMessages(c *websocket.Conn) error {
	roomID := c.Params("roomID") + "action"

	sub := h.chatService.SubscribeToRoom(roomID)

	for {
		go func() {
			for {
				_, _, err := c.ReadMessage()
				if err != nil {
					h.chatService.CloseSubscription(sub)
					c.Close()
					return
				}
			}
		}()

		message, err := sub.ReceiveMessage(context.Background())
		if err != nil {
			h.chatService.CloseSubscription(sub)
			return err
		}

		err = c.WriteMessage(websocket.TextMessage, []byte(message.Payload))
		if err != nil {
			h.chatService.CloseSubscription(sub)
			return err
		}
	}
}
func (h *ChatHandler) RedisCacheGetLastRoomMessages(roomID string) ([]string, error) {

	message, err := h.chatService.RedisCacheGetLastRoomMessages(roomID)
	if err != nil {
		return nil, err
	}
	return message, nil
}
func (h *ChatHandler) Actions(c *fiber.Ctx) error {
	idS := c.Context().UserValue("_id").(string)
	id, err := primitive.ObjectIDFromHex(idS)
	if err != nil {
		return err
	}
	verified := c.Context().UserValue("verified").(bool)

	var req domain.Action
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": err.Error(),
		})
	}
	err = h.chatService.ModeratorRestrictions(req.ActionAgainst, req.Room)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "ModeratorRestrictions",
		})
	}
	if req.Action == "baneado" {
		errBaneado := h.chatService.Baneado(req, id, verified)
		if errBaneado != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "baneado",
			})

		}

	} else if req.Action == "removeban" {
		errRemoveban := h.chatService.Removeban(req, id, verified)
		if errRemoveban != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "removeban",
			})

		}
	} else if req.Action == "vip" {
		errVip := h.chatService.Vip(req, id, verified)
		if errVip != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "Vip",
			})

		}

	} else if req.Action == "rVip" {
		errRemoveVip := h.chatService.RemoveVip(req, id, verified)
		if errRemoveVip != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "removeVip",
			})

		}
	} else if req.Action == "timeOut" {
		errTimeOut := h.chatService.TimeOut(req, id, verified)
		if errTimeOut != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "TimeOut",
			})

		}
	} else if req.Action == "moderator" {
		if errModerator := h.chatService.Moderator(req, id, verified); errModerator != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "Moderator",
			})
		}

	} else if req.Action == "rModerator" {
		if errRemoveModerator := h.chatService.RemoveModerator(req, id, verified); errRemoveModerator != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "RemoveModerator",
			})
		}
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "unrecognized command",
		})
	}
}

func (h *ChatHandler) ActionModerator(c *fiber.Ctx) error {
	NameUser := c.Context().UserValue("nameUser").(string)
	verified := c.Context().UserValue("verified").(bool)

	// request validate
	var req domain.ModeratorAction
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}
	if errModeratorActionValidate := req.ModeratorActionValidate(); errModeratorActionValidate != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": errModeratorActionValidate.Error(),
		})
	}
	//  moderador?
	Moderator, errGetUserInfo := h.chatService.GetUserInfo(req.Room, NameUser, verified)

	if errGetUserInfo != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": "StatusInternalServerError",
		})
	}
	if !Moderator {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "not a moderator",
		})
	}
	// puede hacer acciones contra todos menos contra el streamer
	err := h.chatService.ModeratorRestrictions(req.ActionAgainst, req.Room)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": "ModeratorRestrictions",
		})
	}
	// Action  moderador
	if req.Action == "moderator" {
		errTimeOut := h.chatService.ModeratorActionModerator(req, verified)
		if errTimeOut != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "TimeOut",
			})

		}
	} else if req.Action == "rModerator" {
		errTimeOut := h.chatService.ModeratorActionUnModerator(req, verified)
		if errTimeOut != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "TimeOut",
			})

		}
	} else if req.Action == "vip" {
		errTimeOut := h.chatService.ModeratorActionVip(req, verified)
		if errTimeOut != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "TimeOut",
			})

		}
	} else if req.Action == "rVip" {
		errTimeOut := h.chatService.ModeratorActionunVip(req, verified)
		if errTimeOut != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "TimeOut",
			})

		}
	} else if req.Action == "timeOut" {
		errTimeOut := h.chatService.ModeratorActionTimeOut(req, verified)
		if errTimeOut != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "TimeOut",
			})

		}
	} else if req.Action == "baneado" {
		errBaneado := h.chatService.ModeratorActionBaneado(req, verified)
		if errBaneado != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "baneado",
			})

		}
	} else if req.Action == "removeban" {
		errRemoveban := h.chatService.ModeratorActionRemoveban(req, verified)
		if errRemoveban != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"data": "StatusInternalServerError",
			})
		} else {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"data": "removeban",
			})

		}
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "unrecognized command",
		})
	}
}
func (h *ChatHandler) ActionIdentidadUser(c *fiber.Ctx) error {
	NameUser := c.Context().UserValue("nameUser").(string)
	verified := c.Context().UserValue("verified").(bool)

	// request validate
	var req domain.ActionIdentidadUser
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}

	err := h.chatService.ActionIdentidadUser(req, NameUser, verified)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": "StatusInternalServerError",
		})
	} else {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"data": "TimeOut",
		})

	}

}

// commandos
func (h *ChatHandler) GetCommands(c *fiber.Ctx) error {
	IdUserToken := c.Context().UserValue("_id").(string)
	IdUserTokenP, errinObjectID := primitive.ObjectIDFromHex(IdUserToken)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	Datacommands, err := h.chatService.GetCommands(IdUserTokenP)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":    Datacommands,
		"message": "ok",
	})
}
func (h *ChatHandler) UpdataCommands(c *fiber.Ctx) error {
	IdUserToken := c.Context().UserValue("_id").(string)
	IdUserTokenP, errinObjectID := primitive.ObjectIDFromHex(IdUserToken)
	if errinObjectID != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "StatusInternalServerError",
		})
	}
	var req domain.CommandsUpdata
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}

	if err := req.CommandsUpdataValidata(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": err.Error(),
		})
	}
	err := h.chatService.UpdataCommands(IdUserTokenP, req.CommandsUpdata)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "ok",
	})
}

func (h *ChatHandler) GetInfoUserInRoom(c *fiber.Ctx) error {
	nameUser := c.Context().UserValue("nameUser").(string)
	var req domain.GetInfoUserInRoom
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": "StatusBadRequest",
		})
	}

	InfoUser, err := h.chatService.GetInfoUserInRoom(nameUser, req.GetInfoUserInRoom)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "ok",
		"data":    InfoUser,
	})
}
