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

var log = &logrus.Logger{
	Out:       os.Stdout,
	Formatter: new(logrus.TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.DebugLevel,
}

func initalize() *disgord.Client {
	// Initalize the random number generator
	rand.Seed(time.Now().UTC().UnixNano())
	
	// Read the config file
	file, err := ioutil.ReadFile("config.toml")
	if err != nil {
		fmt.Println("Failed to read config file: ", err)
	}
	
	// Parse the data
	config := Config{}
	err = toml.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Failed to read parse file: ", err)
	}
	
	fmt.Printf("%+v\n", config)
	
	// Do some bitmath
	var intent disgord.Intent
	intent |= disgord.IntentDirectMessages
	intent |= disgord.IntentDirectMessageReactions
	intent |= disgord.IntentDirectMessageTyping
	fmt.Printf("%b\n", intent)
	
	// Make a disgord client based on config
	client := disgord.New(disgord.Config {
		BotToken: config.Discord.BotToken,
//		Intents: intent,
		Logger: log,
		ProjectName: config.Discord.ProjectName,
		Presence: &disgord.UpdateStatusPayload{
			Since: nil,
			Game: nil,
			Status: config.Discord.StatusMessage,
			AFK: false,
		},
	})
	
	return client
}

func messageCallback(session disgord.Session, msgEvt *disgord.MessageCreate) {
	fmt.Println(msgEvt.Message.Content)
	msgEvt.Message.Reply(context.Background(), session, "hey")
}

func isHey(event interface{}) interface{} {
	var msg *disgord.Message
	
	switch t := event.(type) {
		case *disgord.MessageCreate:
			msg = t.Message
		default:
			return nil
	}
	
	if strings.EqualFold(msg.Content, "hey") ||
		strings.EqualFold(msg.Content, "hello") ||
		strings.EqualFold(msg.Content, "hi") {
			return event
		} else {
			return nil
		}
}

func main() {
	// Set things up and connect to discord
	client := initalize()
	defer client.Gateway().StayConnectedUntilInterrupted()
	
	// Print invite link
	inviteURL, err := client.BotAuthorizeURL()
	if err != nil {
		panic(err)
	}
	fmt.Println(inviteURL)
	
	// Create filter to avoid loop where bot responds to it's own messages
	filter, _ := std.NewMsgFilter(context.Background(), client)
//	filter.SetPrefix(config.Discord.Prefix)
	
	// Register basic handler
	client.Gateway().WithMiddleware(filter.NotByBot, isHey).MessageCreate(messageCallback)
}
