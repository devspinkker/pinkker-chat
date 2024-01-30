package application

import (
	"PINKKER-CHAT/config"
	"PINKKER-CHAT/internal/chat/domain"
	"PINKKER-CHAT/internal/chat/infrastructure"
	"encoding/json"
	"errors"
	"time"

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
	if err != nil {
		return "", err
	}
	return message, nil
}

// chat messages
func (s *ChatService) PublishMessageInRoom(roomID primitive.ObjectID, message string, nameUser string, verified bool) error {

	userInfo, err := s.roomRepository.GetUserInfo(roomID, nameUser, verified)
	if err != nil {
		return err
	}
	if userInfo.Baneado {
		return errors.New("baneado")
	}
	nowUserInfoTimeOut := time.Now()
	if !userInfo.TimeOut.Before(nowUserInfoTimeOut) {
		remainingTime := userInfo.TimeOut.Sub(nowUserInfoTimeOut)
		return errors.New("TimeOut: " + remainingTime.String())
	}

	chatMessage := domain.ChatMessage{
		NameUser:         nameUser,
		Color:            userInfo.Color,
		Message:          message,
		Vip:              userInfo.Vip,
		Subscription:     userInfo.Subscription,
		SubscriptionInfo: userInfo.SubscriptionInfo,
		Baneado:          userInfo.Baneado,
		TimeOut:          userInfo.TimeOut,
		Moderator:        userInfo.Moderator,
		EmblemasChat:     userInfo.EmblemasChat,
	}

	chatMessageJSON, err := json.Marshal(chatMessage)
	if err != nil {
		return err
	}
	// saveMessageChan := make(chan error)
	RedisCacheSetLastRoomMessagesAndPublishMessageChan := make(chan error)

	go s.roomRepository.RedisCacheSetLastRoomMessagesAndPublishMessage(roomID, string(chatMessageJSON), RedisCacheSetLastRoomMessagesAndPublishMessageChan)
	// go s.roomRepository.SaveMessageTheUserInRoom(nameUser, roomID, string(chatMessageJSON), saveMessageChan)
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

	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)

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
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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

	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	if userInfo.Moderator == true {
		return nil
	}

	userInfo.Moderator = true
	MODERATOR := config.MODERATOR()

	userInfo.EmblemasChat["Moderator"] = MODERATOR
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	if userInfo.Moderator == false {
		return nil
	}
	userInfo.Moderator = false
	userInfo.EmblemasChat["Moderator"] = ""

	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	err = s.roomRepository.UpdataUserInfo(action.Room, action.ActionAgainst, userInfo)
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
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	err = s.roomRepository.UpdataUserInfo(roomID, action.ActionAgainst, userInfo)
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
	if userInfo.Moderator == true {
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
func (s *ChatService) UserConnectedStream(roomID, command string) error {

	err := s.roomRepository.UserConnectedStream(roomID, command)
	return err
}
func (s *ChatService) GetInfoUserInRoom(nameUser string, GetInfoUserInRoom primitive.ObjectID) (domain.InfoUser, error) {

	InfoUser, UpdataCommandsErr := s.roomRepository.GetInfoUserInRoom(nameUser, GetInfoUserInRoom)
	return InfoUser, UpdataCommandsErr
}
