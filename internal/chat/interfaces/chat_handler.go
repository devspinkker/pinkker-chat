package interfaces

import (
	"PINKKER-CHAT/internal/chat/application"
	"PINKKER-CHAT/internal/chat/domain"
	"PINKKER-CHAT/pkg/utils"
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
	errPublishMessageInRoom := h.chatService.PublishMessageInRoom(room, req.Message, NameUser, verified)
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
func (h *ChatHandler) UserConnectedStream(roomID, commando string) error {

	err := h.chatService.UserConnectedStream(roomID, commando)

	return err
}
func (h *ChatHandler) InfoUserRoomChache(roomID primitive.ObjectID, nameUser string, verified bool) (domain.UserInfo, error) {

	UserInfo, err := h.chatService.InfoUserRoomChache(roomID, nameUser, verified)

	return UserInfo, err
}
func (h *ChatHandler) ReceiveMessageFromRoom(c *websocket.Conn, nameuser string, connectedUsers *utils.ConnectedUsers) error {
	roomID := c.Params("roomID")

	sub := h.chatService.SubscribeToRoom(roomID)

	for {
		go func() {
			for {
				_, _, err := c.ReadMessage()
				if err != nil {

					if connectedUsers.Get(nameuser) && len(nameuser) >= 4 {
						connectedUsers.Set(nameuser, false)
						_ = h.chatService.UserConnectedStream(roomID, "disconnect")
					}
					h.chatService.CloseSubscription(sub)
					c.Close()
					return
				}
			}
		}()

		message, err := sub.ReceiveMessage(context.Background())
		if err != nil {
			if connectedUsers.Get(nameuser) {
				connectedUsers.Set(nameuser, false)
				_ = h.chatService.UserConnectedStream(roomID, "disconnect")
			}
			h.chatService.CloseSubscription(sub)
			return err
		}

		err = c.WriteMessage(websocket.TextMessage, []byte(message.Payload))
		if err != nil {
			if connectedUsers.Get(nameuser) {
				connectedUsers.Set(nameuser, false)
				_ = h.chatService.UserConnectedStream(roomID, "disconnect")
			}
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
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
	if req.Action == "timeOut" {
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
