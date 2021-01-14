package main

import "context"
import "fmt"
import "math/rand"
import "time"
import "os"
import "io/ioutil"
import toml "github.com/pelletier/go-toml"
import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"
import "github.com/andersfylling/disgord/std"
import "github.com/sirupsen/logrus"

// Represents config options related to discord
type General struct {
	ProjectName string
	HelpMessage string
	Prefix string
	DefaultMazeWidth int
	DefaultMazeHeight int
}

// Represents more technical config options
type Technical struct {
	NumWorkers int
	BotToken string
}

// Represents the config file
type Config struct {
	General General
	Technical Technical
}

// Holds global information for the server
type stuff struct {
	config Config
	logger *logrus.Logger
	client *disgord.Client
}

// Sets up disgord and parses config
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
		BotToken: stuff.config.Technical.BotToken,
		// Intents: intent,
		Logger: stuff.logger,
		ProjectName: stuff.config.General.ProjectName,
	})
}

// Listens for messges in the channel and deals with them
func (stuff *stuff) worker(
	helpEvtChan <-chan *disgord.MessageCreate,
	genEvtChan <-chan *disgord.MessageCreate) {
	
	var helpMsg = stuff.config.General.HelpMessage
	
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
				
				// Reply with help
				msg.Reply(context.Background(), stuff.client, helpMsg)
				
			case data, ok := <-genEvtChan:
				if !ok {
					fmt.Println("channel is dead")
					return
				}
				msg = data.Message

				// Get the maze from the message
				coolMaze, err := stuff.getMaze(msg)
				if err != nil {
					msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))

				} else {
					// Reply to the message with the maze
					msg.Reply(context.Background(), stuff.client, "```maze\n" + coolMaze.Stringify() + "```")

					// Read the maze for debugging
					var maze2 *maze.DummyMaze = &maze.DummyMaze{}
					maze2.ReadMaze("```maze\n" + coolMaze.Stringify() + "```")
				}
		}
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
	// BUG(iComputeDaily): No idea how big the buffer should be
	helpEvtChan := make(chan *disgord.MessageCreate, 100)
	genEvtChan := make(chan *disgord.MessageCreate, 100)
	for workerNum := 0; workerNum < stuff.config.Technical.NumWorkers; workerNum++ {
		go stuff.worker(helpEvtChan, genEvtChan)
	}
	
	// Create filter to avoid loop where bot responds to it's own messages
	filter, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filter.SetPrefix(stuff.config.General.Prefix + "maze ")
	
	// Create filter to tell if it's the help command
	filterHelp, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filterHelp.SetPrefix("help")
	
	// Create filter to tell if its a generation request
	filterGen, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filterGen.SetPrefix("gen")
	
	// Create middleware to log messages
	logFilter, _ := std.NewLogFilter(stuff.client)
	
	// Register handler for "help" command
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, // Make shure message isn't by the bot
		std.CopyMsgEvt, // Copy the message so that other handlers don't have problems
		filter.HasPrefix, filter.StripPrefix, // Check if has "!maze"
		filterHelp.HasPrefix, filterHelp.StripPrefix, // Check if has "help"
		logFilter.LogMsg).MessageCreateChan(helpEvtChan) // Log message and push to channel
		
	// Register handler for "generate" command
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, // Make shure message isn't by the bot
		std.CopyMsgEvt, // Copy the message so that other handlers don't have problems
		filter.HasPrefix, filter.StripPrefix, // Check if has "!maze"
		filterGen.HasPrefix, filterGen.StripPrefix, // Check if has "gen"
		logFilter.LogMsg).MessageCreateChan(genEvtChan) // Log message and push to channel
										
}
