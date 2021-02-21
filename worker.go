package main

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"go.uber.org/zap"
	"strings"
	"unicode/utf8"
)

func mentionGetPrefix(msg *disgord.Message) {
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

	// DEBUGGING
	stuff.logger.Debug("Mention recived", zap.String("message", prefixIsMsg))
}

func genCmd(msg *disgord.Message, prefix string) {
	msg.Content = strings.TrimPrefix(msg.Content, "gen")

	// Get the maze from the message
	coolMaze, err := getMaze(msg, prefix)
	if err != nil {
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))
		return
	}

	// Reply to the message with the maze
	msg.Reply(context.Background(), stuff.client, "```maze\n"+coolMaze.Stringify()+"```")
}

func helpCmd(msg *disgord.Message, prefix string) {
	// Replace placeholders in help message
	helpMsg := strings.ReplaceAll(stuff.config.Messages.HelpMessage, "<prefix>", prefix)
	// Reply with help
	msg.Reply(context.Background(), stuff.client, helpMsg)
}

func setPrefixCmd(msg *disgord.Message, prefix string) {
	// Check if user has permission for command
	permBit, err := msg.Member.GetPermissions(context.Background(), stuff.client)
	if err != nil {
		stuff.logger.Error("Failed to get premissions for member!")
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", stuff.config.Messages.GenericError))
		return
	}
	if !permBit.Contains(disgord.PermissionManageServer) {
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", stuff.config.Messages.NoPermsError))
		return
	}

	msg.Content = strings.TrimPrefix(msg.Content, "setPrefix")

	// Set the prefix
	newPrefix, err := setPrefix(msg, prefix)
	if err != nil {
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))
		return
	}

	// Substitute old and new prefixes
	prefixChangeMsg := strings.ReplaceAll(stuff.config.Messages.PrefixChangeMsg, "<oldPrefix>", prefix)
	prefixChangeMsg = strings.ReplaceAll(prefixChangeMsg, "<newPrefix>", newPrefix)

	// Send message to inform user of changes
	msg.Reply(context.Background(), stuff.client, prefixChangeMsg)
}

func invalidCmd(msg *disgord.Message, prefix string) {
	// Find the leftmost command
	cmd := spaceSepRegex.FindString(msg.Content)

	// If the user didn't input a command
	if cmd == "" {
		// Subsitute values
		noCmdError := strings.ReplaceAll(stuff.config.Messages.NoCmdError, "<prefix>", prefix)

		// Reply with message
		msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", noCmdError))
		return
	}

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

			mentionGetPrefix(msg)

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
				genCmd(msg, prefix)

			case strings.HasPrefix(msg.Content, "help"):
				helpCmd(msg, prefix)

			case strings.HasPrefix(msg.Content, "setPrefix"):
				setPrefixCmd(msg, prefix)

			default:
				invalidCmd(msg, prefix)
			}
		}
	}
}
