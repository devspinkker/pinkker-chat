package application

import (
	"PINKKER-CHAT/config"
	"PINKKER-CHAT/internal/chat/domain"
	"PINKKER-CHAT/internal/chat/infrastructure"
	"PINKKER-CHAT/pkg/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatService struct {
	roomRepository *infrastructure.PubSubService
}

func NewChatService(roomRepository *infrastructure.PubSubService) *ChatService {
	return &ChatService{
		roomRepository: roomRepository,
	}
}
func (s *ChatService) GetMessagesForSecond(IdVod primitive.ObjectID, startTime time.Time) ([]domain.ChatMessage, error) {
	return s.roomRepository.GetMessagesForSecond(IdVod, startTime)

}

// acciones los recibir el mensajes
func (s *ChatService) SubscribeToRoom(roomID string) *redis.PubSub {
	sub := s.roomRepository.SubscribeToRoom(roomID)
	return sub
}
func (s *ChatService) CloseSubscription(sub *redis.PubSub) error {
	return s.roomRepository.CloseSubscription(sub)
}

func (s *ChatService) ReceiveMessageFromRoom(roomID string) (string, error) {
	message, err := s.roomRepository.ReceiveMessageFromRoom(roomID)
	return message, err
}
func (s *ChatService) GetWebSocketClientsInRoom(roomID string) ([]*websocket.Conn, error) {
	clients, err := utils.NewChatService().GetWebSocketClientsInRoom(roomID)

	return clients, err
}
func (s *ChatService) FindStreamByStreamer(nameUser string) (domain.Stream, error) {
	stream, err := s.roomRepository.FindStreamByStreamer(nameUser)

	return stream, err
}

// chat messages
func (s *ChatService) PublishMessageInRoom(roomID primitive.ObjectID, message, ResNameUser, ResMsj, nameUser string, verified bool, id primitive.ObjectID) error {

	userInfo, err := s.roomRepository.GetUserInfo(roomID, nameUser, verified)
	if err != nil {
		return err
	}
	VERIFIED := config.PARTNER()
	PRIME := config.PINKKERPRIME()

	currentEmblemasChat := userInfo.EmblemasChat

	if verified {
		currentEmblemasChat["Verified"] = VERIFIED
	} else {
		currentEmblemasChat["Verified"] = ""
	}

	if userInfo.PinkkerPrime {
		currentEmblemasChat["PinkkerPrime"] = PRIME
	} else {
		currentEmblemasChat["PinkkerPrime"] = ""
	}

	userInfo.EmblemasChat = currentEmblemasChat

	if !userInfo.StreamerChannelOwner {
		if userInfo.Baneado {
			return errors.New("baneado")
		}
		nowUserInfoTimeOut := time.Now()
		if !userInfo.TimeOut.Before(nowUserInfoTimeOut) {
			remainingTime := userInfo.TimeOut.Sub(nowUserInfoTimeOut)
			return errors.New("TimeOut: " + remainingTime.String())
		}

		modChat, err := s.roomRepository.RedisGetModStream(roomID)
		if err != nil {
			modChat = ""
		}

		if modChat == "Following" {
			if userInfo.Following.Email == "" {
				return errors.New("only followers")
			}
			Anique, err := s.roomRepository.RedisGetAntiqueStreamDuration(roomID)
			if err != nil {
				return err
			}
			// Convertimos Anique (segundos) a un tiempo en el pasado
			antiqueTime := time.Now().Add(-time.Duration(Anique) * time.Second)

			// Si el usuario empezó a seguir después de la antigüedad requerida, retornar error
			if userInfo.Following.Since.After(antiqueTime) {
				return errors.New("el usuario no cumple con la antigüedad mínima requerida")
			}

		} else if modChat == "Subscriptions" {
			err = s.validateSubscription(userInfo)
			if err != nil {
				return err
			}

		}

		ModSlowModeStream, err := s.roomRepository.RedisGetModSlowModeStream(roomID)
		if err != nil {
			ModSlowModeStream = 0
		}
		allowedTime := userInfo.LastMessage.Add(time.Duration(ModSlowModeStream) * time.Second)
		if time.Now().Before(allowedTime) {
			return errors.New("no puedes enviar un mensaje en este momento")
		}

	}

	chatMessage := domain.ChatMessage{
		NameUser:             nameUser,
		Color:                userInfo.Color,
		Message:              message,
		Vip:                  userInfo.Vip,
		Subscription:         userInfo.Subscription,
		SubscriptionInfo:     userInfo.SubscriptionInfo,
		Baneado:              userInfo.Baneado,
		TimeOut:              userInfo.TimeOut,
		Moderator:            userInfo.Moderator,
		EmblemasChat:         currentEmblemasChat,
		Identidad:            userInfo.Identidad,
		StreamerChannelOwner: userInfo.StreamerChannelOwner,
		PinkkerPrime:         userInfo.PinkkerPrime,
		ResNameUser:          ResNameUser,
		ResMessage:           ResMsj,
		Id:                   primitive.NewObjectID(),
		Timestamp:            time.Now(),
	}

	chatMessageJSON, err := json.Marshal(chatMessage)
	if err != nil {
		return err
	}
	saveMessageChan := make(chan error)
	RedisCacheSetLastRoomMessagesAndPublishMessageChan := make(chan error)

	go s.roomRepository.RedisCacheSetLastRoomMessagesAndPublishMessage(roomID, string(chatMessageJSON), RedisCacheSetLastRoomMessagesAndPublishMessageChan, nameUser)
	go s.roomRepository.RedisCacheAddUniqueUserInteraction(roomID, nameUser, RedisCacheSetLastRoomMessagesAndPublishMessageChan)
	go s.roomRepository.SaveMessageTheUserInRoom(id, roomID, string(chatMessageJSON), saveMessageChan)

	go func() {
		if message[0] == '!' {

			GetCommandsFromCacheerr := s.roomRepository.GetCommandsFromCache(roomID, message)
			if GetCommandsFromCacheerr == redis.Nil {
				err = s.roomRepository.PublishCommandInTheRoom(roomID, message)
			}
		}
	}()

	return nil
}
func (s *ChatService) validateSubscription(userInfo domain.UserInfo) error {
	if userInfo.Subscription == primitive.NilObjectID {
		return errors.New("only subscribers")
	}

	if userInfo.SubscriptionInfo.SubscriptionStart == (time.Time{}) || userInfo.SubscriptionInfo.SubscriptionEnd == (time.Time{}) {
		return errors.New("invalid subscription dates")
	}
	if time.Now().After(userInfo.SubscriptionInfo.SubscriptionEnd) {
		return errors.New("subscription has expired")
	}

	return nil
}
func (s *ChatService) PublishAction(roomID string, noty map[string]interface{}) error {
	return s.roomRepository.PublishAction(roomID, noty)

}
func (s *ChatService) RedisCacheGetLastRoomMessages(roomID string) ([]string, error) {
	message, err := s.roomRepository.RedisCacheGetLastRoomMessages(roomID)
	if err != nil {
		return nil, err
	}
	return message, nil
}
func (s *ChatService) InfoUserRoomChache(roomID primitive.ObjectID, nameUser string, verified bool) (domain.UserInfo, error) {
	UserInfo, err := s.roomRepository.GetUserInfo(roomID, nameUser, verified)
	return UserInfo, err
}

// action
func (s *ChatService) Baneado(action domain.Action, idUser primitive.ObjectID, verified bool) error {

	stream, err := s.roomRepository.GetStreamByIdUser(idUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	if userInfo.Baneado {
		return nil
	}
	userInfo.Baneado = true

	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)

	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) Removeban(action domain.Action, nameUser primitive.ObjectID, verified bool) error {
	stream, err := s.roomRepository.GetStreamByIdUser(nameUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	if !userInfo.Baneado {
		return nil
	}

	userInfo.Baneado = false
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) Vip(action domain.Action, nameUser primitive.ObjectID, verified bool) error {
	stream, err := s.roomRepository.GetStreamByIdUser(nameUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	if userInfo.Vip {
		return nil
	}
	userInfo.Vip = true
	VIP := config.VIP()

	userInfo.EmblemasChat["Vip"] = VIP
	fmt.Println(userInfo.EmblemasChat["Vip"])
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) RemoveVip(action domain.Action, nameUser primitive.ObjectID, verified bool) error {
	stream, err := s.roomRepository.GetStreamByIdUser(nameUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	if !userInfo.Vip {
		return nil
	}

	userInfo.Vip = false
	userInfo.EmblemasChat["Vip"] = ""
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) TimeOut(action domain.Action, nameUser primitive.ObjectID, verified bool) error {
	stream, err := s.roomRepository.GetStreamByIdUser(nameUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	userInfo.TimeOut = time.Now().Add(time.Duration(action.TimeOut) * time.Minute)
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) Moderator(action domain.Action, nameUser primitive.ObjectID, verified bool) error {
	stream, err := s.roomRepository.GetStreamByIdUser(nameUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	if userInfo.Moderator {
		return nil
	}

	userInfo.Moderator = true
	MODERATOR := config.MODERATOR()

	userInfo.EmblemasChat["Moderator"] = MODERATOR
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) RemoveModerator(action domain.Action, nameUser primitive.ObjectID, verified bool) error {
	stream, err := s.roomRepository.GetStreamByIdUser(nameUser)
	if err != nil {
		return err
	}
	roomID := stream.ID
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	if !userInfo.Moderator {
		return nil
	}
	userInfo.Moderator = false
	userInfo.EmblemasChat["Moderator"] = ""

	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

// action del Moderatores
func (s *ChatService) ModeratorActionBaneado(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	if userInfo.Baneado {
		return nil
	}
	userInfo.Baneado = true
	err = s.roomRepository.UpdataUserInfo(action.Room, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}
	return nil
}
func (s *ChatService) ModeratorActionModerator(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room
	fmt.Println("POR")
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	fmt.Println("que")

	if userInfo.Moderator {
		return nil
	}

	userInfo.Moderator = true
	MODERATOR := config.MODERATOR()

	userInfo.EmblemasChat["Moderator"] = MODERATOR
	err = s.roomRepository.UpdataUserInfo(action.Room, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}
	return nil
}
func (s *ChatService) ModeratorActionUnModerator(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	if userInfo.Moderator {
		return nil
	}

	userInfo.Moderator = false
	userInfo.EmblemasChat["Moderator"] = ""
	err = s.roomRepository.UpdataUserInfo(action.Room, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}
	return nil
}
func (s *ChatService) ModeratorActionRemoveban(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room

	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	if !userInfo.Baneado {
		return nil
	}

	userInfo.Baneado = false
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}
func (s *ChatService) ModeratorActionTimeOut(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	userInfo.TimeOut = time.Now().Add(time.Duration(action.TimeOut) * time.Minute)
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChatService) ModeratorActionVip(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}

	if userInfo.Vip {
		return nil
	}
	userInfo.Vip = true
	VIP := config.VIP()

	userInfo.EmblemasChat["Vip"] = VIP
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}
func (s *ChatService) ModeratorActionunVip(action domain.ModeratorAction, verified bool) error {
	roomID := action.Room
	userInfo, err := s.roomRepository.GetUserInfo(roomID, action.ActionAgainst, verified)
	if err != nil {
		return err
	}
	if !userInfo.Vip {
		return nil
	}

	userInfo.Vip = false
	userInfo.EmblemasChat["Vip"] = ""
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}
func (s *ChatService) ActionIdentidadUser(action domain.ActionIdentidadUser, NameUser string, verified bool) error {
	roomID := action.Room

	userInfo, err := s.roomRepository.GetUserInfo(roomID, NameUser, verified)
	if err != nil {
		return err
	}
	// modificarlo al usuariofv
	if action.Color != "" {
		userInfo.Color = action.Color
	}
	if action.Identidad != "" {
		userInfo.Identidad = config.IdentidadSignoZodiacal(action.Identidad)
	}
	if action.Identidad == "" && action.Color == "" {
		userInfo.Identidad = ""
	}
	err = s.roomRepository.UpdataUserInfo(roomID, NameUser, userInfo, verified)
	if err != nil {
		return err
	}

	return nil
}
func (s *ChatService) GetUserInfoStruct(roomID primitive.ObjectID, nameUser string, verified bool) (domain.UserInfo, error) {
	userInfo, errGetUserInfo := s.roomRepository.GetUserInfo(roomID, nameUser, verified)
	return userInfo, errGetUserInfo
}
func (s *ChatService) GetUserInfo(roomID primitive.ObjectID, nameUser string, verified bool) (bool, error) {
	userInfo, errGetUserInfo := s.roomRepository.GetUserInfo(roomID, nameUser, verified)
	if errGetUserInfo != nil {
		return false, errGetUserInfo
	}
	if userInfo.Moderator {
		return true, nil
	} else {
		return false, nil
	}
}

// comandos chat
func (s *ChatService) GetCommands(roomID primitive.ObjectID) (domain.Datacommands, error) {
	Datacommands, err := s.roomRepository.GetCommands(roomID)
	return Datacommands, err
}
func (s *ChatService) UpdataCommands(roomID primitive.ObjectID, newCommands map[string]string) error {

	UpdataCommandsErr := s.roomRepository.UpdataCommands(roomID, newCommands)
	return UpdataCommandsErr
}
func (s *ChatService) UserConnectedStream(roomID, nameUser, action string, id primitive.ObjectID) error {
	ctx := context.TODO()
	err := s.roomRepository.UserConnectedStream(ctx, roomID, nameUser, action, id)
	return err
}

func (s *ChatService) RedisFindMatchingUsersInRoomByPrefix(roomID, nameUser string) ([]string, error) {
	ctx := context.TODO()
	return s.roomRepository.RedisFindMatchingUsersInRoomByPrefix(ctx, roomID, nameUser)
}

func (s *ChatService) SaveMessageAnclarRedis(roomID string, anclarMessage domain.AnclarMessageData) error {
	err := s.roomRepository.SaveMessageAnclarRedis(roomID, anclarMessage)
	return err
}
func (s *ChatService) GetAncladoMessageFromRedis(roomID string) (map[string]interface{}, error) {
	data, err := s.roomRepository.GetAncladoMessageFromRedis(roomID)
	return data, err
}
func (s *ChatService) GetInfoUserInRoom(nameUser string, GetInfoUserInRoom primitive.ObjectID) (domain.InfoUser, error) {

	InfoUser, UpdataCommandsErr := s.roomRepository.GetInfoUserInRoom(nameUser, GetInfoUserInRoom)
	return InfoUser, UpdataCommandsErr
}

func (s *ChatService) ModeratorRestrictions(ActionAgainst string, room primitive.ObjectID) error {

	err := s.roomRepository.ModeratorRestrictions(ActionAgainst, room)
	return err
}
