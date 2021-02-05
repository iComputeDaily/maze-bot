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
import "strings"
import "database/sql"
import _ "github.com/jackc/pgx/v4/stdlib"
import "strconv"

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
	Prefix string
	DefaultMazeWidth int
	DefaultMazeHeight int
}

type Messages struct {
	HelpMessage string
	NoCmdError string
	InvalidCmdError string
	TooManyArgsError string
	UnknownArgError string
	SizeError string
	GenericError string
}

// Represents more technical config options
type Technical struct {
	NumWorkers int
	BotToken string
	DBUrl string
}

// Represents the config file
type Config struct {
	General General
	Messages Messages
	Technical Technical
}

// Holds global information for the server
type things struct {
	config Config
	logger *zap.Logger
	client *disgord.Client
	db *sql.DB
}

var stuff things

// Sets up disgord and parses config
func initalize() {
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

	// Setup the database
	stuff.db, err = sql.Open("pgx", stuff.config.Technical.DBUrl)

	if err != nil {
		stuff.logger.Panic("Failed to establish a database connection!",
		zap.String("URL", stuff.config.Technical.DBUrl), zap.Error(err))
	}
}

// Checks if the message has the costom prefix
func stripCostomPrefixIfExists(event interface{}) interface{} {
	var msg *disgord.Message
	
	// Turn the event into a message
	switch t := event.(type) {
		case *disgord.MessageCreate:
			msg = t.Message
		default:
			// Unless it's not one in witch case cancel
			return nil
	}

	// Has the message prefix
	var prefix string

	// Get prefix from database
	row := stuff.db.QueryRow("SELECT prefix FROM prefixes WHERE guild_id = $1",
		strconv.FormatUint(uint64(msg.GuildID), 10))
	err := row.Scan(&prefix)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			prefix = stuff.config.General.Prefix
		default:
			stuff.logger.Error("Failed to retrive database results!",
				zap.Uint64("guild_id", uint64(msg.GuildID)),
				zap.Error(err))
		}
	}


	if strings.HasPrefix(msg.Content, fmt.Sprint(prefix, "maze")) {
		// Note that disgord.MessageCreate contains only a pointer to the
		// message so dispite asigning the message to our own varible we still
		// modified the event.
		return event
	} else {
		return nil
	}
}

func main() {
	// Set things up and connect to discord
	initalize()
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
	msgEventChan := make(chan *disgord.MessageCreate, 100)
	for workerNum := 0; workerNum < stuff.config.Technical.NumWorkers; workerNum++ {

		go worker(msgEventChan)
	}

	// Create a filter so we can check if the message is by the bot
	filter, _ := std.NewMsgFilter(context.Background(), stuff.client)

	// Register handler for messages with the prefix
	stuff.client.Gateway().WithMiddleware(filter.NotByBot, // Make shure message isn't by the bot
		std.CopyMsgEvt, // Copy the message so that other handlers don't have problems
		stripCostomPrefixIfExists, // Checks to see if the message has the prefix, and if so removes
		).MessageCreateChan(msgEventChan) // Push to channel
}
