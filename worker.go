package main

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"go.uber.org/zap"
	"strings"
	"unicode/utf8"
)

func addMsgFeilds(msg *disgord.Message, logger *zap.SugaredLogger) *zap.SugaredLogger {
	return stuff.logger.With(
		"UsrID", msg.Author.ID,
		"UsrName", msg.Author.Tag(),
		"MsgID", msg.ID,
		"GuildID", msg.GuildID,
		"MsgContent", msg.Content,
	)
}

func mentionGetPrefix(msg *disgord.Message, logger *zap.SugaredLogger) {
	// Get the server prefix
	var prefix string

	if msg.IsDirectMessage() {
		prefix = "!"
	} else {
		prefix = getPrefix(msg.GuildID, stuff.db)
	}

	// Substitue prefix in message
	prefixIsMsg := strings.ReplaceAll(stuff.config.Messages.PrefixIsMsg, "<prefix>", prefix)

	// Reply with prefix
	msg.Reply(context.Background(), stuff.client, prefixIsMsg)

	// Logging
	logger.Infow("Mention recived")
}

func genCmd(msg *disgord.Message, prefix string, logger *zap.SugaredLogger) {
	msg.Content = strings.TrimPrefix(msg.Content, "gen")

	// Get the maze from the message
	coolMaze, logger, err := getMaze(msg, prefix, logger)
	if err != nil {
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))
		return
	}

	// Reply to the message with the maze
	msg.Reply(context.Background(), stuff.client, "```maze\n"+coolMaze.Stringify()+"```")

	// Logging
	logger.Infow("Maze gen request recived")
}

func helpCmd(msg *disgord.Message, prefix string, logger *zap.SugaredLogger) {
	// Reply with help
	helpMsg := strings.ReplaceAll(stuff.config.Messages.HelpMessage, "<prefix>", prefix)
	msg.Reply(context.Background(), stuff.client, helpMsg)

	// Logging
	logger.Infow("Help message request recived")
}

func setPrefixCmd(msg *disgord.Message, prefix string, logger *zap.SugaredLogger) {
	if msg.IsDirectMessage() {
		logger.Infow("Tried to change the prefix in dm")
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", stuff.config.Messages.NewPrefixInDmError))
		return
	}

	// Check if user has permission for command
	permBit, err := msg.Member.GetPermissions(context.Background(), stuff.client)
	if err != nil {
		logger.Errorw("Failed to get premissions for member!")
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", stuff.config.Messages.GenericError))
		return
	}
	if !permBit.Contains(disgord.PermissionManageServer) {
		logger.Infow("Did not have sufficent permissions to change prefix")
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", stuff.config.Messages.NoPermsError))
		return
	}

	msg.Content = strings.TrimPrefix(msg.Content, "setPrefix")

	// Set the prefix
	newPrefix, err := setPrefix(msg, prefix, logger)
	if err != nil {
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))
		return
	}

	// Substitute old and new prefixes
	prefixChangeMsg := strings.ReplaceAll(stuff.config.Messages.PrefixChangeMsg, "<oldPrefix>", prefix)
	prefixChangeMsg = strings.ReplaceAll(prefixChangeMsg, "<newPrefix>", newPrefix)

	// Send message to inform user of changes
	msg.Reply(context.Background(), stuff.client, prefixChangeMsg)

	// Logging
	logger.Infow("Changed prefix", "NewPrefix", newPrefix)
}

func invalidCmd(msg *disgord.Message, prefix string, logger *zap.SugaredLogger) {
	// Find the leftmost command
	cmd := spaceSepRegex.FindString(msg.Content)

	// If the user didn't input a command
	if cmd == "" {
		logger.Infow("No command provided")

		// Reply with message
		noCmdError := strings.ReplaceAll(stuff.config.Messages.NoCmdError, "<prefix>", prefix)
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", noCmdError))
		return
	}

	logger.Infow("Invalid command", "Command", cmd)

	// Subsitute values
	invalidCmdError := strings.ReplaceAll(stuff.config.Messages.InvalidCmdError, "<prefix>", prefix)
	invalidCmdError = strings.ReplaceAll(invalidCmdError, "<command>", cmd)

	// Reply with message
	msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", invalidCmdError))
}

// Listens for messges in the channel and deals with them
func worker(msgEvtChan <-chan *disgord.MessageCreate,
	mentionEventChan <-chan *disgord.MessageCreate) {
	for {
		var msg *disgord.Message

		// Wait for messages
		select {
		// On recived mention
		case data, ok := <-mentionEventChan:
			// Check if channel is dead
			if !ok {
				stuff.logger.Panic("Mention channel is dead!")
			}
			msg = data.Message

			ctxLogger := addMsgFeilds(msg, stuff.logger)
			ctxLogger = stuff.logger.With(
				"Type", "mention",
			)

			mentionGetPrefix(msg, ctxLogger)

		case data, ok := <-msgEvtChan:
			// Get the message
			if !ok {
				stuff.logger.Panic("Invalid channel is dead!")
				return
			}
			msg = data.Message

			// Figure out how big the first rune is
			_, size := utf8.DecodeRuneInString(msg.Content)

			// Get the first rune(the prefix)
			prefix := string(msg.Content[0:size])

			// Trim the prefix
			msg.Content = strings.TrimPrefix(msg.Content, fmt.Sprint(prefix, "maze"))

			// Whitespace might cause problems
			msg.Content = strings.TrimSpace(msg.Content)

			switch {
			case strings.HasPrefix(msg.Content, "gen"):
				ctxLogger := addMsgFeilds(msg, stuff.logger)
				ctxLogger = ctxLogger.With(
					"Prefix", prefix,
					"Type", "genCmd",
				)
				genCmd(msg, prefix, ctxLogger)

			case strings.HasPrefix(msg.Content, "help"):
				ctxLogger := addMsgFeilds(msg, stuff.logger)
				ctxLogger = ctxLogger.With(
					"Prefix", prefix,
					"Type", "helpCmd",
				)
				helpCmd(msg, prefix, ctxLogger)

			case strings.HasPrefix(msg.Content, "setPrefix"):
				ctxLogger := addMsgFeilds(msg, stuff.logger)
				ctxLogger = ctxLogger.With(
					"Prefix", prefix,
					"Type", "setPrefixCmd",
				)
				setPrefixCmd(msg, prefix, ctxLogger)

			default:
				ctxLogger := addMsgFeilds(msg, stuff.logger)
				ctxLogger = stuff.logger.With(
					"Prefix", prefix,
					"Type", "invalidCmd",
				)
				invalidCmd(msg, prefix, ctxLogger)
			}
		}
	}
}
