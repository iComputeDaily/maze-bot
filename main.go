package main

import "strings"
import "context"
import "fmt"
import "math/rand"
import "time"
import "os"
import "io/ioutil"
import toml "github.com/pelletier/go-toml"
// import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"
import "github.com/andersfylling/disgord/std"
import "github.com/sirupsen/logrus"

// Represents config options related to discord
type Discord struct {
	ProjectName string
	BotToken string
	StatusMessage string
	Prefix string
}

// Represents the config file
type Config struct {
	Discord Discord
}

// Holds global information for the server
type stuff struct {
	config Config
	logger *logrus.Logger
	client *disgord.Client
}

func (stuff *stuff) initalize() {
	// Initalize the random number generator
	rand.Seed(time.Now().UTC().UnixNano())
	
	// Read the config file
	file, err := ioutil.ReadFile("config.toml")
	if err != nil {
		fmt.Println("Failed to read config file: ", err)
	}
	
	// Parse the data
	err = toml.Unmarshal(file, &stuff.config)
	if err != nil {
		fmt.Println("Failed to parse file: ", err)
	}
	
	// Setup logging
	stuff.logger = &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
	
	// Do some bitmath
	// var intent disgord.Intent
	// intent |= disgord.IntentDirectMessages
	// intent |= disgord.IntentDirectMessageReactions
	// intent |= disgord.IntentDirectMessageTyping
	// fmt.Printf("%b\n", intent)
	
	// Make a disgord client based on config
	stuff.client = disgord.New(disgord.Config {
		BotToken: stuff.config.Discord.BotToken,
		// Intents: intent,
		Logger: stuff.logger,
		ProjectName: stuff.config.Discord.ProjectName,
		Presence: &disgord.UpdateStatusPayload{
			Since: nil,
			Game: nil,
			Status: stuff.config.Discord.StatusMessage,
			AFK: false,
		},
	})
}

// Replys to messages with hey
func messageCallback(session disgord.Session, msgEvt *disgord.MessageCreate) {
	msgEvt.Message.Reply(context.Background(), session, "hey")
}

// Checks if the event is a message with a common greating
func isHey(event interface{}) interface{} {
	var msg *disgord.Message
	
	// Turn the event into a message
	switch t := event.(type) {
		case *disgord.MessageCreate:
			msg = t.Message
		default:
			// Unless it's not one in witch case cancel
			return nil
	}
	
	// Check if it's a common greating word
	if strings.EqualFold(msg.Content, "hey") ||
		strings.EqualFold(msg.Content, "hello") ||
		strings.EqualFold(msg.Content, "hi") {
			// If so continue
			return event
		} else {
			// Otherwise cancel
			return nil
		}
}

func main() {
	// Set things up and connect to discord
	var stuff stuff
	stuff.initalize()
	defer stuff.client.Gateway().StayConnectedUntilInterrupted()
	
	// Print invite link
	inviteURL, err := stuff.client.BotAuthorizeURL()
	if err != nil {
		panic(err)
	}
	fmt.Println(inviteURL)
	
	// Create filter to avoid loop where bot responds to it's own messages
	filter, _ := std.NewMsgFilter(context.Background(), stuff.client)
	
	// Create middleware to log messages
	logFilter, _ := std.NewLogFilter(stuff.client)
	
	// Register basic handler
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, isHey, logFilter.LogMsg).MessageCreate(messageCallback)
}
