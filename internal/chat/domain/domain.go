package domain

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserInfo struct {
	Room                 primitive.ObjectID
	Color                string
	Vip                  bool
	Verified             bool
	Moderator            bool
	Subscription         primitive.ObjectID
	SubscriptionInfo     SubscriptionInfo
	Baneado              bool
	TimeOut              time.Time
	EmblemasChat         map[string]string
	Following            FollowInfo
	StreamerChannelOwner bool
	LastMessage          time.Time
}
type SubscriptionInfo struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty"`
	SubscriptionNameUser string             `bson:"SubscriptionNameUser"`
	SourceUserID         primitive.ObjectID `bson:"sourceUserID"`
	DestinationUserID    primitive.ObjectID `bson:"destinationUserID"`
	SubscriptionStart    time.Time          `bson:"SubscriptionStart"`
	SubscriptionEnd      time.Time          `bson:"SubscriptionEnd"`
	MonthsSubscribed     int                `bson:"MonthsSubscribed"`
	Notified             bool               `bson:"Notified"`
	Text                 string             `bson:"Text"`
}

type FollowInfo struct {
	Since         time.Time `json:"since" bson:"since"`
	Notifications bool      `json:"notifications" bson:"notifications"`
	Email         string    `json:"Email" bson:"Email"`
}
type InfoUser struct {
	ID       primitive.ObjectID       `bson:"_id,omitempty"`
	Nameuser string                   `bson:"NameUser"`
	Color    string                   `bson:"Color"`
	Rooms    []map[string]interface{} `bson:"Rooms"`
}

type ChatMessage struct {
	NameUser             string             `json:"nameUser"`
	Color                string             `json:"Color" bson:"Color"`
	Message              string             `json:"message"`
	Vip                  bool               `json:"vip"`
	Subscription         primitive.ObjectID `json:"subscription"`
	SubscriptionInfo     SubscriptionInfo
	TimeOut              time.Time          `json:"timeOut"`
	Baneado              bool               `json:"baneado"`
	Moderator            bool               `json:"moderator"`
	EmblemasChat         map[string]string  `json:"EmotesChat"`
	StreamerChannelOwner bool               `json:"StreamerChannelOwner"`
	Id                   primitive.ObjectID `json:"Id"`
}
type Datacommands struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Room     primitive.ObjectID `json:"Room" bson:"Room,omitempty"`
	Color    string             `json:"Color" bson:"Color,omitempty"`
	Commands map[string]string  `json:"Commands"bson:"Commands,omitempty"`
}

// request
type Action struct {
	Action        string             `json:"action"`
	ActionAgainst string             `json:"actionAgainst"`
	TimeOut       int                `json:"timeOut"`
	Room          primitive.ObjectID `json:"room"`
}

func (a *Action) Validate() error {
	if len(a.Action) < 3 || len(a.Action) >= 12 {
		return errors.New("La longitud de 'action' debe ser mayor o igual a 3 caracteres y menor a 12")
	}
	if len(a.ActionAgainst) < 3 || len(a.ActionAgainst) >= 15 {
		return errors.New("La longitud de 'actionAgainst' debe ser mayor o igual a 3 caracteres y menor a 12")
	}
	if a.Action == "TimeOut" {
		if a.TimeOut != 2 {
			return errors.New("TimeOut error")
		}
	}
	return nil
}

type ModeratorAction struct {
	Action        string             `json:"action"`
	ActionAgainst string             `json:"actionAgainst"`
	TimeOut       int                `json:"timeOut"`
	Room          primitive.ObjectID `json:"room"`
}

func (a *ModeratorAction) ModeratorActionValidate() error {

	return nil
}

type MessagesTheSendMessagesRoom struct {
	Message string `json:"message"`
}

func (Message *MessagesTheSendMessagesRoom) MessagesTheSendMessagesRoomValidate() error {

	if len(Message.Message) >= 300 || len(Message.Message) <= 0 {
		return errors.New("La longitud de 'action' debe ser mayor o igual a 3 caracteres y menor a 12")
	}
	return nil
}

type ActivateCommands struct {
	CommandName    string `json:"CommandName"`
	CommandContent string `json:"CommandContent"`
}

func (a *ActivateCommands) ActivateCommandsValidata() error {
	if len(a.CommandName) < 3 || len(a.CommandName) >= 12 {
		return errors.New("La longitud de 'CommandName' debe ser mayor o igual a 3 caracteres y menor a 12")
	}
	if a.CommandName[0] != '!' {
		return errors.New("ComandName debe comenzar con !")
	}

	if len(a.CommandContent) < 3 || len(a.CommandContent) >= 15 {
		return errors.New("La longitud de 'CommandContent' debe ser mayor o igual a 3 caracteres y menor a 12")
	}
	return nil
}

type CommandsUpdata struct {
	CommandsUpdata map[string]string `json:"CommandsUpdata"`
}
type GetInfoUserInRoom struct {
	GetInfoUserInRoom primitive.ObjectID `json:"GetInfoUserInRoom"`
}

func (a *CommandsUpdata) CommandsUpdataValidata() error {
	return nil
}

type Stream struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	StreamerID         primitive.ObjectID `json:"streamerId" bson:"StreamerID"`
	Streamer           string             `json:"streamer" bson:"Streamer"`
	StreamerAvatar     string             `json:"streamer_avatar" bson:"StreamerAvatar,omitempty"`
	ViewerCount        int                `json:"ViewerCount"  bson:"ViewerCount,default:0"`
	Online             bool               `json:"online" bson:"Online,default:false"`
	StreamTitle        string             `json:"stream_title" bson:"StreamTitle"`
	StreamCategory     string             `json:"stream_category" bson:"StreamCategory"`
	ImageCategorie     string             `json:"ImageCategorie" bson:"ImageCategorie"`
	StreamNotification string             `json:"stream_notification" bson:"StreamNotification"`
	StreamTag          []string           `json:"stream_tag"  bson:"StreamTag,default:['Español']"`
	StreamLikes        []string           `json:"stream_likes" bson:"StreamLikes"`
	StreamIdiom        string             `json:"stream_idiom" default:"Español" bson:"StreamIdiom,default:'Español'"`
	StreamThumbnail    string             `json:"stream_thumbnail" bson:"StreamThumbnail"`
	StartDate          time.Time          `json:"start_date" bson:"StartDate"`
	Timestamp          time.Time          `json:"Timestamp" bson:"Timestamp"`
	EmotesChat         map[string]string  `json:"EmotesChat" bson:"EmotesChat"`
	ModChat            string             `json:"ModChat" bson:"ModChat"`
	ModSlowMode        int                `json:"ModSlowMode" bson:"ModSlowMode"`
}

type User struct {
	ID                primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Avatar            string                 `json:"Avatar" default:"https://res.cloudinary.com/pinkker/image/upload/v1680478837/foto_default_obyind.png" bson:"Avatar"`
	FullName          string                 `json:"FullName" bson:"FullName"`
	NameUser          string                 `json:"NameUser" bson:"NameUser"`
	PasswordHash      string                 `json:"passwordHash" bson:"PasswordHash"`
	Pais              string                 `json:"Pais" bson:"Pais"`
	Subscriptions     []primitive.ObjectID   `bson:"Subscriptions"`
	Subscribers       []primitive.ObjectID   `bson:"Subscribers"`
	Clips             []primitive.ObjectID   `bson:"Clips,omitempty"`
	ClipsLikes        []primitive.ObjectID   `bson:"ClipsLikes,omitempty"`
	Ciudad            string                 `json:"Ciudad" bson:"Ciudad"`
	Email             string                 `json:"Email" bson:"Email"`
	EmailConfirmation bool                   `json:"EmailConfirmation" bson:"EmailConfirmation,default:false"`
	Role              int                    `json:"role" bson:"Role,default:0"`
	KeyTransmission   string                 `json:"keyTransmission,omitempty" bson:"KeyTransmission"`
	Biography         string                 `json:"biography" default:"Bienvenido a pinkker! actualiza tu biografía en ajustes de cuenta." bson:"Biography"`
	Look              string                 `json:"look" default:"h_std_cc_3032_7_0-undefined-undefined.ch-215-62-78.hd-180-10.lg-270-110" bson:"Look"`
	LookImage         string                 `json:"lookImage" default:"https://res.cloudinary.com/pinkker/image/upload/v1680478837/foto_default_obyind.png" bson:"LookImage"`
	HeadImage         string                 `json:"headImage" default:"https://res.cloudinary.com/pinkker/image/upload/v1680478837/foto_default_obyind.png" bson:"headImage"`
	Color             string                 `json:"color" bson:"Color"`
	BirthDate         time.Time              `json:"birthDate" bson:"BirthDate"`
	Pixeles           float64                `json:"Pixeles,default:0.0" bson:"Pixeles,default:0.0"`
	CustomAvatar      bool                   `json:"customAvatar,omitempty" bson:"CustomAvatar"`
	CountryInfo       map[string]interface{} `json:"countryInfo,omitempty" bson:"CountryInfo"`
	PinkkerPrime      struct {
		Active bool      `json:"active,omitempty" bson:"Active,omitempty"`
		Date   time.Time `json:"date,omitempty" bson:"Date,omitempty"`
	} `json:"pinkkerPrime,omitempty" bson:"PinkkerPrime"`
	Suscribers    []string `json:"suscribers,omitempty" bson:"Suscribers"`
	SocialNetwork struct {
		Facebook  string `json:"facebook,omitempty" bson:"facebook"`
		Twitter   string `json:"twitter,omitempty" bson:"twitter"`
		Instagram string `json:"instagram,omitempty" bson:"instagram"`
		Youtube   string `json:"youtube,omitempty" bson:"youtube"`
		Tiktok    string `json:"tiktok,omitempty" bson:"tiktok"`
	} `json:"socialnetwork,omitempty" bson:"socialnetwork"`
	Cmt                      string                            `json:"cmt,omitempty" bson:"Cmt"`
	Verified                 bool                              `json:"verified,omitempty" bson:"Verified"`
	Website                  string                            `json:"website,omitempty" bson:"Website"`
	Phone                    string                            `json:"phone,omitempty" bson:"Phone"`
	Sex                      string                            `json:"sex,omitempty" bson:"Sex"`
	Situation                string                            `json:"situation,omitempty" bson:"Situation"`
	UserFriendsNotifications int                               `json:"userFriendsNotifications,omitempty" bson:"UserFriendsNotifications"`
	Following                map[primitive.ObjectID]FollowInfo `json:"Following" bson:"Following"`
	Followers                map[primitive.ObjectID]FollowInfo `json:"Followers" bson:"Followers"`
	Timestamp                time.Time                         `json:"Timestamp" bson:"Timestamp"`
	Likes                    []primitive.ObjectID              `json:"Likes" bson:"Likes"`
	Wallet                   string                            `json:"Wallet" bson:"Wallet"`
	Online                   bool                              `json:"Online,omitempty" bson:"Online,omitempty" default:"false"`
}
