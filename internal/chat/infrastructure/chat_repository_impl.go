package infrastructure

import (
	"PINKKER-CHAT/config"
	"PINKKER-CHAT/internal/chat/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// uso general sobre informacion de un usuario en una sala
func (r *PubSubService) GetUserInfo(roomID primitive.ObjectID, nameUser string, verified bool) (domain.UserInfo, error) {
	var userInfo domain.UserInfo
	var infoUser domain.InfoUser

	colors := []string{
		"#b9d6f6", "#e9113c", "#1475e1", "#00ccb3", "#75fd46",
	}

	randomIndex := rand.Intn(len(colors))
	randomColor := colors[randomIndex]
	var InsertuserInfoCollection bool = false

	defaultUserFields := map[string]interface{}{
		"Room":             roomID,
		"Color":            randomColor,
		"Vip":              false,
		"Verified":         verified,
		"Moderator":        false,
		"Subscription":     primitive.ObjectID{},
		"SubscriptionInfo": domain.SubscriptionInfo{},
		"Baneado":          false,
		"TimeOut":          time.Now(),
		"EmblemasChat": map[string]string{
			"Vip":       "",
			"Moderator": "",
			"Verified":  "",
		},
		"Following":            domain.FollowInfo{},
		"StreamerChannelOwner": false,
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
		userInfo.StreamerChannelOwner, _ = storedUserFields["StreamerChannelOwner"].(bool)

		subscriptionValue, _ := storedUserFields["Subscription"].(string)

		subscriptionID, err := primitive.ObjectIDFromHex(subscriptionValue)
		if err == nil {
			userInfo.Subscription = subscriptionID
		} else {
			userInfo.Subscription = primitive.NilObjectID
		}

		subscriptionInfoInterface, ok := storedUserFields["SubscriptionInfo"]
		if ok {
			// Verificar que sea un mapa antes de intentar convertirlo
			if subscriptionInfoMap, ok := subscriptionInfoInterface.(map[string]interface{}); ok {
				subscriptionInfo := domain.SubscriptionInfo{
					ID:                   primitive.NilObjectID,
					SubscriptionNameUser: subscriptionInfoMap["SubscriptionNameUser"].(string),
					SourceUserID:         primitive.NilObjectID,
					DestinationUserID:    primitive.NilObjectID,
				}

				// Verificar la existencia y tipo de los campos antes de convertirlos
				if startStr, ok := subscriptionInfoMap["SubscriptionStart"].(string); ok {
					startTime, err := time.Parse(time.RFC3339, startStr)
					if err == nil {
						subscriptionInfo.SubscriptionStart = startTime
					}
				}

				if endStr, ok := subscriptionInfoMap["SubscriptionEnd"].(string); ok {
					endTime, err := time.Parse(time.RFC3339, endStr)
					if err == nil {
						subscriptionInfo.SubscriptionEnd = endTime
					}
				}

				if months, ok := subscriptionInfoMap["MonthsSubscribed"].(float64); ok {
					subscriptionInfo.MonthsSubscribed = int(months)
				}

				if notified, ok := subscriptionInfoMap["Notified"].(bool); ok {
					subscriptionInfo.Notified = notified
				}

				if text, ok := subscriptionInfoMap["Text"].(string); ok {
					subscriptionInfo.Text = text
				}

				userInfo.SubscriptionInfo = subscriptionInfo
			} else {
				userInfo.SubscriptionInfo = domain.SubscriptionInfo{}
			}
		} else {
			userInfo.SubscriptionInfo = domain.SubscriptionInfo{}
		}

		if followingInfoMap, ok := storedUserFields["Following"].(map[string]interface{}); ok {
			followingInfo := domain.FollowInfo{}

			if sinceTime, ok := followingInfoMap["Since"].(time.Time); ok {
				followingInfo.Since = sinceTime
			}

			if notificationsBool, ok := followingInfoMap["Notifications"].(bool); ok {
				followingInfo.Notifications = notificationsBool
			}

			if emailString, ok := followingInfoMap["Email"].(string); ok {
				followingInfo.Email = emailString
			}

			userInfo.Following = followingInfo
		} else {
			userInfo.Following = domain.FollowInfo{}
		}
		userInfo.Moderator = storedUserFields["Moderator"].(bool)
		userInfo.Baneado = storedUserFields["Baneado"].(bool)
		if colorValue, ok := storedUserFields["Color"]; ok && colorValue != nil {
			userInfo.Color = colorValue.(string)
		} else {
			userInfo.Color = "blue"
		}
		userInfo.Verified = verified
		if verified {
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
			InsertuserInfoCollection = true
			// deberia creaar la info del chat en cuanto se subscribe

		} else if err != nil {
			return userInfo, err
		}
		roomExists := false
		for _, room := range infoUser.Rooms {
			if room["Room"] == roomID {
				fmt.Println("x1111")
				fmt.Println(infoUser.Rooms)
				fmt.Println("x1111")
				roomExists = true
				valor, ok := room["EmblemasChat"].(map[string]interface{})
				if !ok {
					fmt.Println("La clave 'EmblemasChat' no tiene el tipo de mapa esperado.")
				}

				userInfo = domain.UserInfo{
					Room:      roomID,
					Color:     randomColor,
					Vip:       room["Vip"].(bool),
					Moderator: room["Moderator"].(bool),
					Verified:  room["Verified"].(bool),
					Baneado:   room["Baneado"].(bool),
				}
				fmt.Println("x33")
				fmt.Println(room)
				fmt.Println("x33")
				fmt.Println("x21")
				fmt.Println(room["Following"])
				fmt.Println("x21")
				followingInfoMap, ok := room["Following"].(map[string]interface{})
				if ok {
					followingInfo := domain.FollowInfo{}

					if sinceTime, ok := followingInfoMap["since"].(time.Time); ok {
						followingInfo.Since = sinceTime
					}

					if notificationsBool, ok := followingInfoMap["notifications"].(bool); ok {
						followingInfo.Notifications = notificationsBool
					}

					if emailString, ok := followingInfoMap["Email"].(string); ok {
						followingInfo.Email = emailString
					}

					userInfo.Following = followingInfo
				} else {
					// userInfo.Following = domain.FollowInfo{}
					fmt.Println("El campo 'Following' no tiene el tipo de mapa esperado.")
				}

				if owner, ok := room["StreamerChannelOwner"].(bool); ok {
					userInfo.StreamerChannelOwner = owner
				} else {
					streamerChannelOwner, _ := r.streamerChannelOwner(nameUser, roomID)
					userInfo.StreamerChannelOwner = streamerChannelOwner
				}
				subscriptionID, ok := room["Subscription"].(primitive.ObjectID)
				if !ok {
					subscriptionID = primitive.NilObjectID
				}
				userInfo.Subscription = subscriptionID
				userInfo.Subscription = subscriptionID
				if TimeOutInterface, ok := room["TimeOut"]; ok {
					if TimeOutdAgo, ok := TimeOutInterface.(time.Time); ok {
						userInfo.TimeOut = TimeOutdAgo
					} else {
						userInfo.TimeOut = time.Now()
					}
				} else {
					userInfo.TimeOut = time.Now()
				}
				if verified {
					VERIFIED := config.PARTNER()
					userInfo.EmblemasChat = map[string]string{
						"Vip":       valor["Vip"].(string),
						"Moderator": valor["Moderator"].(string),
						"Verified":  VERIFIED,
					}
				} else {
					userInfo.EmblemasChat = map[string]string{
						"Vip":       valor["Vip"].(string),
						"Moderator": valor["Moderator"].(string),
						"Verified":  valor["Verified"].(string),
					}
				}

				// Buscar el documento de suscripción usando el ID de Subscription
				subscription, err := r.getSubscriptionByID(subscriptionID)

				if err == nil {
					// Se encontró el documento de suscripción
					subscriptionInfo := domain.SubscriptionInfo{
						ID:                   subscription.ID,
						SubscriptionNameUser: subscription.SubscriptionNameUser,
						SourceUserID:         subscription.SourceUserID,
						DestinationUserID:    subscription.DestinationUserID,
						SubscriptionStart:    subscription.SubscriptionStart,
						SubscriptionEnd:      subscription.SubscriptionEnd,
						MonthsSubscribed:     subscription.MonthsSubscribed,
						Notified:             subscription.Notified,
						Text:                 subscription.Text,
					}

					userInfo.SubscriptionInfo = subscriptionInfo
				} else if err == mongo.ErrNoDocuments {
					userInfo.SubscriptionInfo = domain.SubscriptionInfo{}
				} else {
					userInfo.SubscriptionInfo = domain.SubscriptionInfo{}
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
				Room:         roomID,
				Vip:          false,
				Color:        randomColor,
				Moderator:    false,
				Verified:     verified,
				Subscription: primitive.ObjectID{},
				Baneado:      false,
				TimeOut:      time.Now(),
				EmblemasChat: map[string]string{
					"Vip":       "",
					"Moderator": "",
					"Verified":  "",
				},
				SubscriptionInfo: domain.SubscriptionInfo{},
				Following:        domain.FollowInfo{},
			}
			streamerChannelOwner, _ := r.streamerChannelOwner(nameUser, roomID)
			userInfo.StreamerChannelOwner = streamerChannelOwner
			if verified {
				VERIFIED := config.PARTNER()
				userInfo.EmblemasChat = map[string]string{
					"Vip":       "",
					"Moderator": "",
					"Verified":  VERIFIED,
				}
			}

			newRoom := map[string]interface{}{
				"Room":                 roomID,
				"Vip":                  false,
				"Color":                randomColor,
				"Moderator":            false,
				"Verified":             verified,
				"Subscription":         primitive.ObjectID{},
				"Baneado":              false,
				"TimeOut":              time.Now(),
				"EmblemasChat":         userInfo.EmblemasChat,
				"Following":            domain.FollowInfo{},
				"StreamerChannelOwner": userInfo.StreamerChannelOwner,
			}

			infoUser.Rooms = append(infoUser.Rooms, newRoom)
			_, err := Collection.UpdateOne(context.Background(), filter, bson.M{"$set": infoUser})
			if err != nil {
				return domain.UserInfo{}, err
			}

		}
		if InsertuserInfoCollection {

			userInfoCollection := domain.InfoUser{
				Nameuser: nameUser,
				Color:    randomColor,
				Rooms:    []map[string]interface{}{defaultUserFields},
			}
			_, err = Collection.InsertOne(context.Background(), userInfoCollection)
			if err != nil {
				return domain.UserInfo{}, err
			}
		}
		err = r.RedisCacheSetUserInfo(userHashKey, userInfo)
		if err != nil {
			return domain.UserInfo{}, err
		}
	}

	return userInfo, nil
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
	roomCommands.Color = "#7c7ce1"
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
		NameUser:     "PinkkerBot",
		Message:      commandContent,
		Color:        "#7c7ce1",
		Vip:          true,
		Subscription: primitive.ObjectID{},
		Baneado:      false,
		TimeOut:      time.Now(),
		Moderator:    false,
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
		NameUser:     "PinkkerBot",
		Message:      commandContent,
		Color:        "blue",
		Vip:          true,
		Subscription: primitive.ObjectID{},
		Baneado:      false,
		TimeOut:      time.Now(),
		Moderator:    false,
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

	pipeline.LTrim(context.Background(), Room.Hex()+"LastMessages", 0, 19)

	pipeline.Publish(context.Background(), Room.Hex(), message).Err()

	_, err := pipeline.Exec(context.Background())
	if err != nil {
		RedisCacheSetLastRoomMessagesChan <- err
	}

	RedisCacheSetLastRoomMessagesChan <- nil
}
func (r *PubSubService) RedisGetModStream(Room primitive.ObjectID) (string, error) {

	value, err := r.redisClient.Get(context.Background(), Room.Hex()).Result()
	return value, err

}

func (r *PubSubService) getSubscriptionByID(subscriptionID primitive.ObjectID) (domain.SubscriptionInfo, error) {
	collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("Subscriptions")

	var subscription domain.SubscriptionInfo
	filter := bson.M{"_id": subscriptionID}
	err := collection.FindOne(context.Background(), filter).Decode(&subscription)

	return subscription, err
}
func (r *PubSubService) GetStreamByIdUser(IdUser primitive.ObjectID) (domain.Stream, error) {
	collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("Streams")

	var Stream domain.Stream
	filter := bson.M{"StreamerID": IdUser}
	err := collection.FindOne(context.Background(), filter).Decode(&Stream)

	return Stream, err
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
	messages, err := r.redisClient.LRange(context.Background(), Room+"LastMessages", 0, 24).Result()
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
			"Rooms.$.Vip":          userInfo.Vip,
			"Rooms.$.Moderator":    userInfo.Moderator,
			"Rooms.$.Baneado":      userInfo.Baneado,
			"Rooms.$.TimeOut":      userInfo.TimeOut,
			"Rooms.$.EmblemasChat": userInfo.EmblemasChat,
		},
	}
	_, err := Collection.UpdateOne(context.Background(), filter, updateFields)
	if err != nil {
		return err
	}

	userHashKey := "userInformation:" + nameUser + ":inTheRoom:" + roomID.Hex()
	userFields := map[string]interface{}{
		"Vip":              userInfo.Vip,
		"Baneado":          userInfo.Baneado,
		"TimeOut":          userInfo.TimeOut,
		"Moderator":        userInfo.Moderator,
		"EmblemasChat":     userInfo.EmblemasChat,
		"Color":            userInfo.Color,
		"SubscriptionInfo": userInfo.SubscriptionInfo,
		"Subscription":     userInfo.Subscription,
		"Verified":         userInfo.Verified,
		"Room":             userInfo.Room,
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
func (r *PubSubService) GetCommands(id primitive.ObjectID) (domain.Datacommands, error) {
	collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("Streams")

	var Stream domain.Stream
	filter := bson.M{"StreamerID": id}
	err := collection.FindOne(context.Background(), filter).Decode(&Stream)
	if err != nil {
		return domain.Datacommands{}, err
	}
	Collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("CommandsInChat")

	filter = bson.M{"Room": Stream.ID}
	var Commands domain.Datacommands
	err = Collection.FindOne(context.Background(), filter).Decode(&Commands)

	if err == mongo.ErrNoDocuments {
		// El documento no existe, crearlo
		defaultCommands := domain.Datacommands{
			Room:     Stream.ID,
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

func (r *PubSubService) UpdataCommands(id primitive.ObjectID, newCommands map[string]string) error {
	collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("Streams")

	var Stream domain.Stream
	filter := bson.M{"StreamerID": id}
	err := collection.FindOne(context.Background(), filter).Decode(&Stream)
	if err != nil {
		return err
	}

	cachedCommandsKey := "CommandsInChatThe:" + Stream.ID.Hex()
	_, err = r.redisClient.Del(context.Background(), cachedCommandsKey).Result()
	if err != nil {
		return err
	}
	Collection := r.MongoClient.Database("PINKKER-BACKEND").Collection("CommandsInChat")
	filter = bson.M{"Room": Stream.ID}
	update := bson.M{
		"$set": bson.M{
			"Commands": newCommands,
		},
	}
	_, UpdateOneerr := Collection.UpdateOne(context.Background(), filter, update)
	return UpdateOneerr
}
func (r *PubSubService) GetInfoUserInRoom(nameUser string, GetInfoUserInRoom primitive.ObjectID) (domain.InfoUser, error) {
	database := r.MongoClient.Database("PINKKER-BACKEND")
	var InfoUser domain.InfoUser
	err := database.Collection("UserInformationInAllRooms").FindOne(
		context.Background(),
		bson.M{"NameUser": nameUser, "Rooms.Room": GetInfoUserInRoom},
	).Decode(&InfoUser)
	return InfoUser, err
}
func (r *PubSubService) UserConnectedStream(roomID, command string) error {
	database := r.MongoClient.Database("PINKKER-BACKEND")
	streamCollection := database.Collection("Streams")
	categoriaCollection := database.Collection("Categorias")

	streamUpdate := bson.M{"$inc": bson.M{"ViewerCount": 1}}

	if command == "disconnect" {
		streamUpdate = bson.M{"$inc": bson.M{"ViewerCount": -1}}
	}

	roomIDObj, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		return err
	}

	streamFilter := bson.M{"_id": roomIDObj}

	var updatedStream domain.Stream
	err = streamCollection.FindOneAndUpdate(context.Background(), streamFilter, streamUpdate).Decode(&updatedStream)
	if err != nil {
		return err
	}

	categoria := updatedStream.StreamCategory

	categoriaUpdate := bson.M{"$inc": bson.M{"Spectators": 1}}

	if command == "disconnect" {
		categoriaUpdate = bson.M{"$inc": bson.M{"Spectators": -1}}
	}

	categoriaFilter := bson.M{"Name": categoria}

	result, err := categoriaCollection.UpdateOne(context.Background(), categoriaFilter, categoriaUpdate)

	if result.MatchedCount == 0 {
		nuevaCategoria := bson.M{
			"Name":       categoria,
			"Img":        "",
			"Spectators": 1,
			"Tags":       []string{},
		}
		_, err = categoriaCollection.InsertOne(context.Background(), nuevaCategoria)
	}

	return err
}
func (r *PubSubService) ModeratorRestrictions(ActionAgainst string, room primitive.ObjectID) error {
	database := r.MongoClient.Database("PINKKER-BACKEND")

	var Streamer domain.Stream

	err := database.Collection("Streams").FindOne(context.TODO(), bson.M{"_id": room}).Decode(&Streamer)
	if err != nil {
		return err
	}

	if Streamer.Streamer == ActionAgainst {
		return errors.New("ModeratorRestrictions, no se puede banear al streamer")
	}
	return nil
}
func (r *PubSubService) streamerChannelOwner(nameUser string, room primitive.ObjectID) (bool, error) {
	db := r.MongoClient.Database("PINKKER-BACKEND")
	StreamsCollection := db.Collection("Streams")
	filter := bson.M{"_id": room}
	var infoStream domain.Stream
	err := StreamsCollection.FindOne(context.Background(), filter).Decode(&infoStream)
	if err != nil {
		return false, err
	}
	coincide := false
	if infoStream.Streamer == nameUser {
		coincide = true
	}
	return coincide, nil
}
