package infrastructure

import (
	"PINKKER-CHAT/config"
	"PINKKER-CHAT/internal/chat/domain"
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PubSubService struct {
	redisClient   *redis.Client
	MongoClient   *mongo.Client
	subscriptions map[string]*redis.PubSub
}

func NewRepository(redisClient *redis.Client, MongoClient *mongo.Client) *PubSubService {
	return &PubSubService{
		redisClient:   redisClient,
		MongoClient:   MongoClient,
		subscriptions: map[string]*redis.PubSub{},
	}
}

// acciones los recibir el mensajes
func (s *PubSubService) SubscribeToRoom(roomID string) *redis.PubSub {
	sub := s.redisClient.Subscribe(context.Background(), roomID)
	return sub
}

func (s *PubSubService) CloseSubscription(sub *redis.PubSub) error {
	return sub.Close()
}

func (s *PubSubService) ReceiveMessageFromRoom(roomID string) (string, error) {
	sub := s.redisClient.Subscribe(context.Background(), roomID)
	defer sub.Close()

	for {
		msg, err := sub.ReceiveMessage(context.Background())
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}

func (p *PubSubService) PublishCommandInTheRoom(roomID primitive.ObjectID, commandName string) error {

	Collection := p.MongoClient.Database("PINKKER-BACKEND").Collection("CommandsInChat")
	filter := bson.M{"Room": roomID}

	var roomCommands domain.Datacommands
	roomCommands.Color = "blue"
	err := Collection.FindOne(context.Background(), filter).Decode(&roomCommands)
	if err != nil {
		return err
	}

	commandContent, exists := roomCommands.Commands[commandName]
	if !exists {
		return nil
	}

	VERIFIED := config.PARTNER()
	chatMessage := domain.ChatMessage{
		NameUser:      "PinkkerBot",
		Message:       commandContent,
		Color:         "blue",
		Vip:           true,
		Subscription:  "",
		SubscribedAgo: "",
		Baneado:       false,
		TimeOut:       time.Now(),
		Moderator:     false,
		EmblemasChat: map[string]string{
			"Vip":       "",
			"Moderator": "",
			"Verified":  VERIFIED,
		},
	}
	chatMessageJSON, err := json.Marshal(chatMessage)
	if err != nil {
		return err
	}
	err = p.redisClient.Publish(
		context.Background(),
		roomID.Hex(),
		string(chatMessageJSON),
	).Err()
	if err != nil {
		return err
	}

	commandsJSON, err := json.Marshal(roomCommands)
	if err != nil {
		return err
	}

	cacheExpiration := 5 * time.Minute
	redisClientErr := p.redisClient.Set(
		context.Background(),
		"CommandsInChatThe:"+roomID.Hex(),
		commandsJSON,
		cacheExpiration,
	).Err()

	return redisClientErr
}

func (p *PubSubService) GetCommandsFromCache(roomID primitive.ObjectID, commandName string) error {
	cachedCommands, err := p.redisClient.Get(context.Background(), "CommandsInChatThe:"+roomID.Hex()).Result()
	if err != nil {
		return err
	}
	var commandsJSON domain.Datacommands
	err = json.Unmarshal([]byte(cachedCommands), &commandsJSON)
	if err != nil {
		return err
	}
	commandContent, exists := commandsJSON.Commands[commandName]
	if !exists {
		return nil
	}

	VERIFIED := config.PARTNER()
	chatMessage := domain.ChatMessage{
		NameUser:      "PinkkerBot",
		Message:       commandContent,
		Color:         "blue",
		Vip:           true,
		Subscription:  "",
		SubscribedAgo: "",
		Baneado:       false,
		TimeOut:       time.Now(),
		Moderator:     false,
		EmblemasChat: map[string]string{
			"Vip":       "",
			"Moderator": "",
			"Verified":  VERIFIED,
		},
	}
	chatMessageJSON, err := json.Marshal(chatMessage)
	if err != nil {
		return err
	}

	err = p.redisClient.Publish(
		context.Background(),
		roomID.Hex(),
		string(chatMessageJSON),
	).Err()
	if err != nil {
		return err
	}

	return nil
}

// acciones de enviar el mensaje
func (s *PubSubService) SaveMessageTheUserInRoom(nameUser string, roomID primitive.ObjectID, message string, saveMessageChan chan error) {
	nSave := s.redisClient.HSet(context.Background(), "messageFromUser:"+nameUser+":inTheRoom:"+roomID.Hex(), time.Now().UnixNano(), message)
	if nSave.Err() != nil {
		saveMessageChan <- nSave.Err()
	}
	saveMessageChan <- nil
}
func (r *PubSubService) RedisCacheSetLastRoomMessagesAndPublishMessage(Room primitive.ObjectID, message string, RedisCacheSetLastRoomMessagesChan chan error) {

	pipeline := r.redisClient.Pipeline()

	pipeline.LPush(context.Background(), Room.Hex()+"LastMessages", message)

	pipeline.LTrim(context.Background(), Room.Hex()+"LastMessages", 0, 9)

	pipeline.Publish(context.Background(), Room.Hex(), message).Err()

	_, err := pipeline.Exec(context.Background())
	if err != nil {
		RedisCacheSetLastRoomMessagesChan <- err
	}

	RedisCacheSetLastRoomMessagesChan <- nil
}

// uso general sobre informacion de un usuario en una sala
func (r *PubSubService) GetUserInfo(roomID primitive.ObjectID, nameUser string, verified bool) (domain.UserInfo, error) {
	var userInfo domain.UserInfo
	var infoUser domain.InfoUser
	colors := []string{
		"red", "blue", "green", "yellow", "orange",
		"pink", "purple", "turquoise", "gray", "white",
	}

	randomIndex := rand.Intn(len(colors))

	// Obtener el color aleatorio
	randomColor := colors[randomIndex]
	defaultUserFields := map[string]interface{}{
		"Room":          roomID,
		"Color":         randomColor,
		"Vip":           false,
		"Verified":      verified,
		"Moderator":     false,
		"Subscription":  "inactive",
		"SubscribedAgo": "",
		"Baneado":       false,
		"TimeOut":       time.Now(),
		"EmblemasChat": map[string]string{
			"Vip":       "",
			"Moderator": "",
			"Verified":  "",
		},
	}

	userHashKey := "userInformation:" + nameUser + ":inTheRoom:" + roomID.Hex()
	userFieldsJSON, err := r.RedisCacheGet(userHashKey)
	if err == nil {
		var storedUserFields map[string]interface{}
		errUnmarshal := json.Unmarshal([]byte(userFieldsJSON), &storedUserFields)
		if errUnmarshal != nil {
			return userInfo, errUnmarshal
		}

		userInfo.Vip, _ = storedUserFields["Vip"].(bool)
		userInfo.Subscription = storedUserFields["Subscription"].(string)
		userInfo.SubscribedAgo = storedUserFields["SubscribedAgo"].(string)
		userInfo.Moderator = storedUserFields["Moderator"].(bool)
		userInfo.Baneado = storedUserFields["Baneado"].(bool)
		userInfo.Color = randomColor
		userInfo.Verified = verified
		if verified == true {
			VERIFIED := config.PARTNER()
			defaultUserFields["EmblemasChat"] = map[string]string{
				"Vip":       "",
				"Moderator": "",
				"Verified":  VERIFIED,
			}
		}
		emblemasChatInterface, ok := storedUserFields["EmblemasChat"].(map[string]interface{})

		if ok {
			userInfo.EmblemasChat = make(map[string]string)
			for key, value := range emblemasChatInterface {
				userInfo.EmblemasChat[key] = value.(string)
			}
		}

		timeStr := storedUserFields["TimeOut"].(string)
		timeOut, errtimeOut := time.Parse(time.RFC3339, timeStr)
		if errtimeOut != nil {
			return userInfo, errtimeOut
		}
		userInfo.TimeOut = timeOut
	} else if err != redis.Nil {
		return userInfo, err
	} else {

		Collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("UserInformationInAllRooms")
		filter := bson.M{"NameUser": nameUser}
		err = Collection.FindOne(context.Background(), filter).Decode(&infoUser)

		if err == mongo.ErrNoDocuments {

			userInfoCollection := domain.InfoUser{
				NameUser: nameUser,
				Color:    randomColor,
				Rooms:    []map[string]interface{}{defaultUserFields},
			}
			userInfo = domain.UserInfo{
				Room:          roomID,
				Color:         randomColor,
				Vip:           false,
				Verified:      verified,
				Moderator:     false,
				Subscription:  "",
				SubscribedAgo: "",
				Baneado:       false,
				TimeOut:       time.Now(),
				EmblemasChat: map[string]string{
					"Vip":       "",
					"Moderator": "",
					"Verified":  "",
				},
			}
			if verified == true {
				VERIFIED := config.PARTNER()
				defaultUserFields["EmblemasChat"] = map[string]string{
					"Vip":       "",
					"Moderator": "",
					"Verified":  VERIFIED,
				}
				userInfo.EmblemasChat = map[string]string{
					"Vip":       "",
					"Moderator": "",
					"Verified":  VERIFIED,
				}
			}
			_, err = Collection.InsertOne(context.Background(), userInfoCollection)
			if err != nil {
				return domain.UserInfo{}, err
			}
		} else if err != nil {
			return userInfo, err
		} else {
			roomExists := false
			for _, room := range infoUser.Rooms {
				if room["Room"] == roomID {
					roomExists = true
					userInfo = domain.UserInfo{
						Room:          roomID,
						Color:         randomColor,
						Vip:           false,
						Moderator:     false,
						Verified:      verified,
						Subscription:  "inactive",
						SubscribedAgo: "",
						Baneado:       false,
						TimeOut:       time.Now(),
						EmblemasChat: map[string]string{
							"Vip":       "",
							"Moderator": "",
							"Verified":  "",
						},
					}
					if verified == true {
						VERIFIED := config.PARTNER()
						userInfo.EmblemasChat = map[string]string{
							"Vip":       "",
							"Moderator": "",
							"Verified":  VERIFIED,
						}
					}
					err = r.RedisCacheSetUserInfo(userHashKey, userInfo)
					if err != nil {
						return domain.UserInfo{}, err
					}
					return userInfo, err
				}
			}
			if !roomExists {
				userInfo = domain.UserInfo{
					Room:          roomID,
					Vip:           false,
					Color:         randomColor,
					Moderator:     false,
					Verified:      verified,
					Subscription:  "inactive",
					SubscribedAgo: "",
					Baneado:       false,
					TimeOut:       time.Now(),
					EmblemasChat: map[string]string{
						"Vip":       "",
						"Moderator": "",
						"Verified":  "",
					},
				}
				if verified == true {
					VERIFIED := config.PARTNER()
					userInfo.EmblemasChat = map[string]string{
						"Vip":       "",
						"Moderator": "",
						"Verified":  VERIFIED,
					}
				}
				newRoom := map[string]interface{}{
					"Room":          roomID,
					"Vip":           false,
					"Color":         randomColor,
					"Moderator":     false,
					"Verified":      verified,
					"Subscription":  "inactive",
					"SubscribedAgo": "",
					"Baneado":       false,
					"TimeOut":       time.Now(),
					"EmblemasChat":  userInfo.EmblemasChat,
				}

				infoUser.Rooms = append(infoUser.Rooms, newRoom)
				_, err := Collection.UpdateOne(context.Background(), filter, bson.M{"$set": infoUser})
				if err != nil {
					return domain.UserInfo{}, err
				}
			}
		}

		err = r.RedisCacheSetUserInfo(userHashKey, userInfo)
		if err != nil {
			return domain.UserInfo{}, err
		}
	}
	return userInfo, nil
}

func (r *PubSubService) RedisCacheSetUserInfo(userHashKey string, userInfo domain.UserInfo) error {
	userFieldsJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}
	cacheExpiration := 10 * time.Minute
	err = r.redisClient.Set(context.Background(), userHashKey, userFieldsJSON, cacheExpiration).Err()
	return err
}

// guardar los ultimos 10 mensajes enviandos a un chat
func (r *PubSubService) RedisCacheGetLastRoomMessages(Room string) ([]string, error) {
	messages, err := r.redisClient.LRange(context.Background(), Room+"LastMessages", 0, 9).Result()
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// updata User
func (r *PubSubService) UpdataUserInfo(roomID primitive.ObjectID, nameUser string, userInfo domain.UserInfo) error {
	Collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("UserInformationInAllRooms")
	filter := bson.M{"NameUser": nameUser, "Rooms.Room": roomID}
	updateFields := bson.M{
		"$set": bson.M{
			"Rooms.$.Room":          userInfo.Room,
			"Rooms.$.Vip":           userInfo.Vip,
			"Rooms.$.Moderator":     userInfo.Moderator,
			"Rooms.$.Subscription":  userInfo.Subscription,
			"Rooms.$.SubscribedAgo": userInfo.SubscribedAgo,
			"Rooms.$.Baneado":       userInfo.Baneado,
			"Rooms.$.TimeOut":       userInfo.TimeOut,
			"Rooms.$.EmblemasChat":  userInfo.EmblemasChat,
		},
	}
	_, err := Collection.UpdateOne(context.Background(), filter, updateFields)
	if err != nil {
		return err
	}

	userHashKey := "userInformation:" + nameUser + ":inTheRoom:" + roomID.Hex()
	userFields := map[string]interface{}{
		"Vip":           userInfo.Vip,
		"Subscription":  userInfo.Subscription,
		"SubscribedAgo": userInfo.SubscribedAgo,
		"Baneado":       userInfo.Baneado,
		"TimeOut":       userInfo.TimeOut,
		"Moderator":     userInfo.Moderator,
		"EmblemasChat":  userInfo.EmblemasChat,
	}

	err = r.RedisCacheSetUpdata(userHashKey, userFields)

	if err != nil {
		return err
	}

	return nil
}
func (r *PubSubService) RedisCacheSetUpdata(userHashKey string, userFields map[string]interface{}) error {
	userFieldsJSON, err := json.Marshal(userFields)
	if err != nil {
		return err
	}
	cacheExpiration := 10 * time.Minute
	err = r.redisClient.Set(context.Background(), userHashKey, userFieldsJSON, cacheExpiration).Err()
	return err
}
func (r *PubSubService) RedisCacheGet(userHashKey string) (string, error) {
	userFieldsJSON, err := r.redisClient.Get(context.Background(), userHashKey).Result()
	return userFieldsJSON, err
}

// comandos Updata
func (r *PubSubService) GetCommands(roomID primitive.ObjectID) (domain.Datacommands, error) {
	Collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("CommandsInChat")
	filter := bson.M{"Room": roomID}
	var Commands domain.Datacommands
	err := Collection.FindOne(context.Background(), filter).Decode(&Commands)

	if err == mongo.ErrNoDocuments {
		// El documento no existe, crearlo
		defaultCommands := domain.Datacommands{
			Room:     roomID,
			Commands: make(map[string]string),
		}

		_, errInsert := Collection.InsertOne(context.Background(), defaultCommands)
		if errInsert != nil {
			return domain.Datacommands{}, errInsert
		}

		return defaultCommands, nil
	} else if err != nil {
		return domain.Datacommands{}, err
	}

	return Commands, nil
}

func (r *PubSubService) UpdataCommands(roomID primitive.ObjectID, newCommands map[string]string) error {
	cachedCommandsKey := "CommandsInChatThe:" + roomID.Hex()
	_, err := r.redisClient.Del(context.Background(), cachedCommandsKey).Result()
	if err != nil {
		return err
	}
	Collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("CommandsInChat")
	filter := bson.M{"Room": roomID}
	update := bson.M{
		"$set": bson.M{
			"Commands": newCommands,
		},
	}
	_, UpdateOneerr := Collection.UpdateOne(context.Background(), filter, update)
	return UpdateOneerr
}
