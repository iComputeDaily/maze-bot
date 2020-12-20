package main

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
	HelpMessage string
	Prefix string
	NumWorkers int
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
		os.Exit(1)
	}
	
	// Parse the data
	err = toml.Unmarshal(file, &stuff.config)
	if err != nil {
		fmt.Println("Failed to parse file: ", err)
		os.Exit(1)
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
	})
}

// Listens for messges in the channel and deals with them
func worker(session disgord.Session, helpEvtChan <-chan *disgord.MessageCreate, helpMsg string) {
	for {
		var msg *disgord.Message
		
		// Wait for messages
		select {
			case data, ok := <-helpEvtChan:
				if !ok {
					fmt.Println("channel is dead")
					return
				}
				msg = data.Message
		}
		// Reply with help
		msg.Reply(context.Background(), session, helpMsg)
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
	
	// Set up channels and workers
	helpEvtChan := make(chan *disgord.MessageCreate, 100) // BUG(iComputeDaily): No idea how big the buffer should be
	for workerNum := 0; workerNum < stuff.config.Discord.NumWorkers; workerNum++ {
		go worker(stuff.client, helpEvtChan, stuff.config.Discord.HelpMessage)
	}
	
	// Create filter to avoid loop where bot responds to it's own messages
	filter, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filter.SetPrefix(stuff.config.Discord.Prefix + "maze ")
	
	// Create filter to tell if it's the help command
	filterHelp, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filterHelp.SetPrefix("help")
	
	// Create middleware to log messages
	logFilter, _ := std.NewLogFilter(stuff.client)
	
	// Register handler for "help" command
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, // Make shure message isn't by the bot
		std.CopyMsgEvt, // Copy the message so that other handlers don't have problems
		filter.HasPrefix, filter.StripPrefix, // Check if has "!maze"
		filterHelp.HasPrefix, filterHelp.StripPrefix, // Check if has "help"
		logFilter.LogMsg).MessageCreateChan(helpEvtChan) // Log message and push to channel
}
