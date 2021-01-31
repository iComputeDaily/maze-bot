package main

import "context"
import "regexp"
import "fmt"
import "math/rand"
import "time"
import "os"
import "io/ioutil"
import toml "github.com/pelletier/go-toml"
import "github.com/andersfylling/disgord"
import "github.com/andersfylling/disgord/std"
import "go.uber.org/zap"
import "encoding/json"

// Regex to match maze type
var isTypeRegex *regexp.Regexp
// Regex to match size
var isSizeRegex *regexp.Regexp
// Regex to match command
var isCmdRegex *regexp.Regexp
// Regrex to match seperations by space
var spaceSepRegex *regexp.Regexp

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
	logger *zap.Logger
	client *disgord.Client
}

// Sets up disgord and parses config
func (stuff *stuff) initalize() {
	// Initalize the random number generator
	rand.Seed(time.Now().UTC().UnixNano())
	
	// Read the config file
	file, err := ioutil.ReadFile("config.toml")
	if err != nil {
		fmt.Println("Failed to read config file:", err)
		os.Exit(1)
	}
	
	// Parse the data
	err = toml.Unmarshal(file, &stuff.config)
	if err != nil {
		fmt.Println("Failed to parse file:", err)
		os.Exit(1)
	}
	
	// Setup logging
	rawJSON := []byte(`{
			"level": "info",
			"encoding": "json",
			"development": false,
			"outputPaths": ["stdout"],
			"encoderConfig": {
				"timeEncoder": "rfc3339",
				"levelEncoder": "capital",
				"durationEncoder": "string",
				"callerEncoder": "short",

				"callerKey":      "CALLER",
				"nameKey":        "NAME",
				"timeKey":        "TIME",
				"levelKey":       "LVL",
				"messageKey":     "MSG",
				"stacktraceKey":  "STACKTRACE"
			}
		}`)

	var logCfg zap.Config
	if err := json.Unmarshal(rawJSON, &logCfg); err != nil {
		fmt.Println("failed to parse logger config:", err)
		os.Exit(1)
	}

	stuff.logger, err = logCfg.Build()
	if err != nil {
		fmt.Println("Failed to create logger:", err)
		os.Exit(1)
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
		Logger: stuff.logger.Sugar(),
		ProjectName: stuff.config.General.ProjectName,
	})

	// Set up regular expressions
	isTypeRegex = regexp.MustCompile(`^(?i)spikey|windy|loopy$`)
	isSizeRegex = regexp.MustCompile(`^(?i)-?\d+x-?\d+$`)
	isCmdRegex = regexp.MustCompile(`^ gen|help`)
	spaceSepRegex = regexp.MustCompile(`\S+`)
}

// Checks if the event isn't valid
func isNotValid(event interface{}) interface{} {
	var msg *disgord.Message
	
	// Turn the event into a message
	switch t := event.(type) {
		case *disgord.MessageCreate:
			msg = t.Message
		default:
			// Unless it's not one in witch case cancel
			return nil
	}

	// Check if it's a valid comand
	if !isCmdRegex.MatchString(msg.Content) {
			// If not continue
			return event
		} else {
			// If so cancel
			return nil
		}
}

func main() {
	// Set things up and connect to discord
	var stuff stuff
	stuff.initalize()
	defer stuff.client.Gateway().StayConnectedUntilInterrupted()
	defer stuff.logger.Sync()
	
	// Print invite link
	inviteURL, err := stuff.client.BotAuthorizeURL()
	if err != nil {
		stuff.logger.DPanic("Failed to create invite link!", zap.Error(err))
	}
	stuff.logger.Info("Invite url", zap.String("URL", inviteURL.String()))

	// Set up channels and workers
	// BUG(iComputeDaily): No idea how big the buffer should be
	helpEvtChan := make(chan *disgord.MessageCreate, 100)
	genEvtChan := make(chan *disgord.MessageCreate, 100)
	invalidEventChan := make(chan *disgord.MessageCreate, 100)

	for workerNum := 0; workerNum < stuff.config.Technical.NumWorkers; workerNum++ {
		go stuff.worker(helpEvtChan, genEvtChan, invalidEventChan)
	}
	
	// Create filter to avoid loop where bot responds to it's own messages
	filter, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filter.SetPrefix(stuff.config.General.Prefix + "maze ")
	
	// Same as previous but without space
	filterNoSpace, _ := std.NewMsgFilter(context.Background(), stuff.client)
	filterNoSpace.SetPrefix(stuff.config.General.Prefix + "maze")

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
		filter.HasPrefix, filter.StripPrefix, // Check if has "!maze "
		filterHelp.HasPrefix, filterHelp.StripPrefix, // Check if has "help"
		).MessageCreateChan(helpEvtChan) // Log message and push to channel
		
	// Register handler for "generate" command
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, // Make shure message isn't by the bot
		std.CopyMsgEvt, // Copy the message so that other handlers don't have problems
		filter.HasPrefix, filter.StripPrefix, // Check if has "!maze "
		filterGen.HasPrefix, filterGen.StripPrefix, // Check if has "gen"
		).MessageCreateChan(genEvtChan) // Push to channel

	// Register handler for only "!maze"
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, // Make shure message isn't by the bot
		std.CopyMsgEvt, // Copy the message so that other handlers don't have problems
		// Check if has "!maze", and log message if so
		// Note that this will log will log all messeges with "!maze" because we have only checked for that so far
		filterNoSpace.HasPrefix, logFilter.LogMsg, filterNoSpace.StripPrefix,
		isNotValid, // Check to see is the message isn't a valid command
		).MessageCreateChan(invalidEventChan) // Push to channel
										
}
