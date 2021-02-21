package main

import (
	// Random stuff
	"context"
	"io/ioutil"
	"math/rand"
	"time"

	// Discord
	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"

	// Text prossesing
	"fmt"
	"golang.org/x/text/unicode/norm"
	"regexp"
	"strings"

	// Logging
	"encoding/json"
	"go.uber.org/zap"

	// Disk storage libs
	"database/sql"
	_ "github.com/jackc/pgx/v4/stdlib"
	toml "github.com/pelletier/go-toml"
)

// Regular expressions
var (
	isTypeRegex   *regexp.Regexp
	isSizeRegex   *regexp.Regexp
	isCmdRegex    *regexp.Regexp
	spaceSepRegex *regexp.Regexp
)

// Config options related to discord
type General struct {
	ProjectName       string
	Prefix            string
	StatusMessage     string
	DefaultMazeWidth  int
	DefaultMazeHeight int
}

// Various messages for the discord bot to output
type Messages struct {
	HelpMessage        string
	PrefixChangeMsg    string
	NoCmdError         string
	InvalidCmdError    string
	TooManyArgsError   string
	UnknownArgError    string
	SizeError          string
	GenericError       string
	PrefixTypeError    string
	PrefixLegnthError  string
	PrefixIsMsg        string
	NewPrefixInDmError string
	NoPermsError       string
}

// More technical config options
type Technical struct {
	NumWorkers int
	BotToken   string
	DBUrl      string
}

// The config file
type Config struct {
	General   General
	Messages  Messages
	Technical Technical
}

// Global information for the server
type things struct {
	config Config
	logger *zap.Logger
	client *disgord.Client
	db     *sql.DB
}

var stuff things

// Sets up disgord and parses config
func initalize() {
	// Initalize the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// Read the config file
	file, err := ioutil.ReadFile("config.toml")
	if err != nil {
		panic(fmt.Sprint("Failed to read config file: ", err))
	}

	// Parse the data
	err = toml.Unmarshal(file, &stuff.config)
	if err != nil {
		panic(fmt.Sprint("Failed to parse file: ", err))
	}

	// Setup logging
	rawJSON := []byte(`{
			"level": "debug",
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
		panic(fmt.Sprint("failed to parse logger config: ", err))
	}

	stuff.logger, err = logCfg.Build()
	if err != nil {
		panic(fmt.Sprint("Failed to create logger: ", err))
	}

	// Setup intent bitmask
	var intent disgord.Intent
	intent |= disgord.IntentDirectMessages
	// intent |= disgord.IntentDirectMessageReactions
	// intent |= disgord.IntentDirectMessageTyping

	// Make a disgord client based on config
	stuff.client = disgord.New(disgord.Config{
		BotToken:    stuff.config.Technical.BotToken,
		Intents:     intent,
		Logger:      stuff.logger.Sugar(),
		ProjectName: stuff.config.General.ProjectName,
	})

	// Update status after 20 seconds(hopefully discord is connected by then)
	go func() {
		time.Sleep(20 * time.Second)
		err = stuff.client.UpdateStatusString(stuff.config.General.StatusMessage)
		if err != nil {
			stuff.logger.DPanic("Failed to set the bots status!", zap.Error(err))
		}
	}()

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
	var prefix string

	// Turn the event into a message
	msgEvt, ok := event.(*disgord.MessageCreate)
	if !ok {
		stuff.logger.Error("A non message made it's way into the message channel.")
		return nil
	}
	msg = msgEvt.Message

	// Get the servers prefix
	if msg.IsDirectMessage() {
		prefix = "!"
	} else {
		prefix = getPrefix(msg.GuildID, stuff.db)
	}

	// Normalize the prefix, and message to hopefully avoid unicode problems*crosses fingers*
	msg.Content = norm.NFC.String(msg.Content)
	prefix = norm.NFC.String(fmt.Sprint(prefix, "maze"))

	// Actualy return whats apropriate
	if strings.HasPrefix(msg.Content, prefix) {
		return event
	} else {
		return nil
	}
}

func main() {
	// Set things up
	initalize()
	defer stuff.db.Close()
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
	mentionEventChan := make(chan *disgord.MessageCreate, 100)
	for workerNum := 0; workerNum < stuff.config.Technical.NumWorkers; workerNum++ {
		go worker(msgEventChan, mentionEventChan)
	}

	// Create a filter
	filter, _ := std.NewMsgFilter(context.Background(), stuff.client)
	logFilter, _ := std.NewLogFilter(stuff.client) // Temp; add logging later.

	// Register handler for messages that mention the bot
	stuff.client.Gateway().WithMiddleware(filter.NotByBot,
		// So that other handlers don't have problems
		std.CopyMsgEvt,
		logFilter.LogMsg,
		filter.HasBotMentionPrefix,
	).MessageCreateChan(mentionEventChan)

	// Register handler for messages with the prefix
	stuff.client.Gateway().WithMiddleware(filter.NotByBot,
		// So that other handlers don't have problems
		std.CopyMsgEvt,
		stripCostomPrefixIfExists,
	).MessageCreateChan(msgEventChan)

	// Connect to discord and wait for something to halpen before we exit
	stuff.client.Gateway().StayConnectedUntilInterrupted()
}
