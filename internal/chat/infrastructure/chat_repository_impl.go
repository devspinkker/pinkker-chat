package infrastructure

import (
	"PINKKER-CHAT/config"
	"PINKKER-CHAT/internal/chat/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
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
func (r *PubSubService) RedisGetModSlowModeStream(Room primitive.ObjectID) (int, error) {
	value, err := r.redisClient.Get(context.Background(), Room.Hex()+"ModSlowMode").Result()
	if err == nil {
		modSlowMode, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}
		return modSlowMode, nil
	} else if err != redis.Nil {
		return 0, err
	}

	var stream domain.Stream
	err = r.MongoClient.Database("PINKKER-BACKEND").Collection("Streams").
		FindOne(context.Background(), bson.M{"_id": Room}).
		Decode(&stream)
	if err != nil {
		return 0, err
	}

	modSlowMode := stream.ModSlowMode

	err = r.redisClient.Set(context.Background(), Room.Hex()+"ModSlowMode", modSlowMode, 200*time.Second).Err()
	if err != nil {
		return 0, err
	}

	return modSlowMode, nil
}

func (r *PubSubService) UserExists(ctx context.Context, nameUser string) (bool, error) {
	userCollection := r.MongoClient.Database("PINKKER-BACKEND").Collection("Users")

	var result domain.User
	err := userCollection.FindOne(ctx, bson.M{"NameUser": nameUser}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *PubSubService) UserConnectedStream(ctx context.Context, roomID, nameUser string, action string) error {
	session, err := r.MongoClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	err = session.StartTransaction()
	if err != nil {
		return err
	}

	// Verificar si el usuario existe
	exists, err := r.UserExists(ctx, nameUser)
	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}
	if !exists {
		return errors.New("user does not exist") // Usuario no encontrado
	}

	err = r.performUserTransaction(ctx, session, roomID, nameUser, action)

	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}

	err = session.CommitTransaction(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *PubSubService) performUserTransaction(ctx context.Context, session mongo.Session, roomID, nameUser string, action string) error {
	activeUsersKey := "ActiveUsers"
	activeUserRoomsKey := "ActiveUserRooms:" + nameUser // Clave para las salas activas del usuario

	// Verificar si el usuario ya está activo en la sala actual
	isActive, err := r.redisClient.SIsMember(ctx, activeUserRoomsKey, roomID).Result()
	if err != nil {
		return err
	}

	// Si la acción es conectar y el usuario ya está activo, no hacer nada
	if action == "connect" && isActive {
		return nil
	}

	// Si la acción es desconectar y el usuario ya está desconectado, no hacer nada
	if action == "disconnect" && !isActive {
		return nil
	}

	// Si la acción es desconectar y el usuario está activo, desconectarlo
	if action == "disconnect" && isActive {
		_, err := r.redisClient.SRem(ctx, activeUserRoomsKey, roomID).Result()
		if err != nil {
			return err
		}

		// Disminuir el contador de espectadores para la sala
		roomIDObj, err := primitive.ObjectIDFromHex(roomID)
		if err != nil {
			return err
		}

		err = r.updateViewerCount(ctx, session, roomIDObj, -1)
		if err != nil {
			return err
		}

		return nil
	}

	// Si la acción es conectar y el usuario está desconectado, conectarlo
	if action == "connect" && !isActive {
		// Agregar al usuario como activo en la nueva sala
		err = r.redisClient.SAdd(ctx, activeUserRoomsKey, roomID).Err()
		if err != nil {
			return err
		}

		// Incrementar el contador de espectadores para la sala
		roomIDObj, err := primitive.ObjectIDFromHex(roomID)
		if err != nil {
			return err
		}

		err = r.updateViewerCount(ctx, session, roomIDObj, 1)
		if err != nil {
			return err
		}

		// Agregar al usuario como activo globalmente si no lo estaba
		err = r.redisClient.SAdd(ctx, activeUsersKey, nameUser).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PubSubService) updateViewerCount(ctx context.Context, session mongo.Session, roomID primitive.ObjectID, delta int) error {
	streamCollection := session.Client().Database("PINKKER-BACKEND").Collection("Streams")
	categoriaCollection := session.Client().Database("PINKKER-BACKEND").Collection("Categorias")

	// Obtener el Stream actual
	var updatedStream domain.Stream
	err := streamCollection.FindOne(ctx, bson.M{"_id": roomID}).Decode(&updatedStream)
	if err != nil {
		return err
	}

	// Verificar que el nuevo ViewerCount no sea negativo
	newViewerCount := updatedStream.ViewerCount + delta
	if newViewerCount < 0 {
		return fmt.Errorf("ViewerCount cannot be negative")
	}

	// Actualizar contador de espectadores para la sala
	_, err = streamCollection.UpdateOne(ctx,
		bson.M{"_id": roomID},
		bson.M{"$inc": bson.M{"ViewerCount": delta}})
	if err != nil {
		return err
	}

	// Obtener la categoría de la sala
	categoria := updatedStream.StreamCategory

	// Obtener la categoría actual
	var updatedCategory domain.Categoria
	err = categoriaCollection.FindOne(ctx, bson.M{"Name": categoria}).Decode(&updatedCategory)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	// Verificar que el nuevo Spectators no sea negativo
	newSpectators := updatedCategory.Spectators + delta
	if newSpectators < 0 {
		return fmt.Errorf("spectators cannot be negative")
	}

	// Actualizar contador de espectadores para la categoría
	if err == mongo.ErrNoDocuments {
		_, err = categoriaCollection.InsertOne(ctx, bson.M{
			"Name":       categoria,
			"Img":        "",
			"Spectators": delta,
			"Tags":       []string{},
		})
	} else {
		_, err = categoriaCollection.UpdateOne(ctx,
			bson.M{"Name": categoria},
			bson.M{"$inc": bson.M{"Spectators": delta}})
	}
	if err != nil {
		return err
	}

	return nil
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
	streamerChannelOwner, _ := r.streamerChannelOwner(nameUser, roomID)
	defaultUserFields := map[string]interface{}{
		"Room":             roomID,      // primitive.ObjectID
		"Color":            randomColor, //string
		"Vip":              false,
		"Verified":         verified, // bool
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
		"Identidad":            "",
		"Following":            domain.FollowInfo{},
		"StreamerChannelOwner": streamerChannelOwner, //bool
		"LastMessage":          time.Now(),
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
		userInfo.Identidad = storedUserFields["Identidad"].(string)

		timeStr := storedUserFields["TimeOut"].(string)
		timeOut, errtimeOut := time.Parse(time.RFC3339, timeStr)
		if errtimeOut != nil {
			return userInfo, errtimeOut
		}
		userInfo.TimeOut = timeOut
		if storedUserFields["LastMessage"] != nil {
			LastMessageStr := storedUserFields["LastMessage"].(string)
			LastMessagOut, LastMessagOutErr := time.Parse(time.RFC3339, LastMessageStr)
			if LastMessagOutErr != nil {
				return userInfo, LastMessagOutErr
			}
			userInfo.LastMessage = LastMessagOut
		} else {
			userInfo.LastMessage = time.Now()
		}
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
				roomExists = true
				valor, ok := room["EmblemasChat"].(map[string]interface{})
				if !ok {
					fmt.Println("La clave 'EmblemasChat' no tiene el tipo de mapa esperado.")
				}

				userInfo = domain.UserInfo{
					Room:      roomID,
					Color:     room["Color"].(string),
					Vip:       room["Vip"].(bool),
					Moderator: room["Moderator"].(bool),
					Verified:  room["Verified"].(bool),
					Baneado:   room["Baneado"].(bool),
				}
				if identidad, ok := room["Identidad"].(string); ok {
					userInfo.Identidad = identidad
				} else {
					userInfo.Identidad = ""

				}
				userInfo.LastMessage = time.Now()
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
					userInfo.Following = domain.FollowInfo{}
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
				LastMessage:      time.Now(),
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
				"LastMessage":          time.Now(),
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
func (p *PubSubService) FindStreamByStreamer(NameUser string) (domain.Stream, error) {

	Collection := p.MongoClient.Database("PINKKER-BACKEND").Collection("Streams")
	filter := bson.M{"Streamer": NameUser}

	var Stream domain.Stream
	err := Collection.FindOne(context.Background(), filter).Decode(&Stream)
	return Stream, err
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
func (p *PubSubService) PublishAction(roomID string, noty map[string]interface{}) error {

	chatMessageJSON, err := json.Marshal(noty)
	if err != nil {
		return err
	}
	err = p.redisClient.Publish(
		context.Background(),
		roomID,
		string(chatMessageJSON),
	).Err()
	if err != nil {
		return err
	}

	return err
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
func (r *PubSubService) RedisCacheSetLastRoomMessagesAndPublishMessage(Room primitive.ObjectID, message string, RedisCacheSetLastRoomMessagesChan chan error, NameUser string) {

	pipeline := r.redisClient.Pipeline()

	pipeline.LPush(context.Background(), Room.Hex()+"LastMessages", message)

	pipeline.LTrim(context.Background(), Room.Hex()+"LastMessages", 0, 19)

	pipeline.Publish(context.Background(), Room.Hex(), message).Err()

	_, err := pipeline.Exec(context.Background())
	if err != nil {
		RedisCacheSetLastRoomMessagesChan <- err
	}
	userHashKey := "userInformation:" + NameUser + ":inTheRoom:" + Room.Hex()

	userInfoStr, err := r.RedisCacheGet(userHashKey)
	if err != nil {
		RedisCacheSetLastRoomMessagesChan <- err
	}

	var userInfo domain.UserInfo
	if err := json.Unmarshal([]byte(userInfoStr), &userInfo); err != nil {
		RedisCacheSetLastRoomMessagesChan <- err
	}

	userInfo.LastMessage = time.Now()

	err = r.RedisCacheSetUserInfo(userHashKey, userInfo)
	RedisCacheSetLastRoomMessagesChan <- err
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
			"Rooms.$.Identidad":    userInfo.Identidad,
			"Rooms.$.Color":        userInfo.Color,
		},
	}
	_, err := Collection.UpdateOne(context.Background(), filter, updateFields)
	if err != nil {
		return err
	}
	streamerChannelOwner, _ := r.streamerChannelOwner(nameUser, roomID)

	userHashKey := "userInformation:" + nameUser + ":inTheRoom:" + roomID.Hex()
	userFields := map[string]interface{}{
		"Vip":                  userInfo.Vip,
		"Baneado":              userInfo.Baneado,
		"TimeOut":              userInfo.TimeOut,
		"Moderator":            userInfo.Moderator,
		"EmblemasChat":         userInfo.EmblemasChat,
		"Color":                userInfo.Color,
		"Identidad":            userInfo.Identidad,
		"SubscriptionInfo":     userInfo.SubscriptionInfo,
		"Subscription":         userInfo.Subscription,
		"Verified":             userInfo.Verified,
		"Room":                 userInfo.Room,
		"LastMessage":          userInfo.LastMessage,
		"StreamerChannelOwner": streamerChannelOwner,
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

func (r *PubSubService) SaveMessageAnclarRedis(roomID string, anclarMessage domain.AnclarMessageData) error {
	ctx := context.Background()
	client := r.redisClient

	jsonData, err := json.Marshal(anclarMessage)
	if err != nil {
		return err
	}

	key := "Anclado:" + roomID

	_, err = client.Do(ctx, "SET", key, jsonData).Result()
	if err != nil {
		return err
	}

	_, err = client.Do(ctx, "EXPIRE", key, int64(2*60)).Result()
	if err != nil {
		return err
	}

	return nil
}

func (r *PubSubService) GetAncladoMessageFromRedis(roomID string) (map[string]interface{}, error) {
	ctx := context.Background()
	client := r.redisClient

	key := "Anclado:" + roomID

	jsonData, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
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
