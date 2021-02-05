package main

import "github.com/andersfylling/disgord"
import "context"
import "fmt"
import "strings"

// Listens for messges in the channel and deals with them
func worker(msgEvtChan <-chan *disgord.MessageCreate) {
	for {
		var msg *disgord.Message
		
		// Wait for messages
		select {
			case data, ok := <-msgEvtChan:
				// Get the message
				if !ok {
					stuff.logger.Panic("Invalid channel is dead!")
					return
				}
				msg = data.Message

				// Get the prefix from the message
				prefix := string(msg.Content[0])

				// Trim the prefix
				msg.Content = strings.TrimPrefix(msg.Content, fmt.Sprint(prefix, "maze"))

				// Whitespace might cause problems
				msg.Content = strings.TrimSpace(msg.Content)

				switch {
					case strings.HasPrefix(msg.Content, "gen"):
						msg.Content = strings.TrimPrefix(msg.Content, "gen")

						// Get the maze from the message
						coolMaze, err := getMaze(msg, prefix)
						if err != nil {
							msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))

						} else {
							// Reply to the message with the maze
							msg.Reply(context.Background(), stuff.client, "```maze\n" + coolMaze.Stringify() + "```")
						}

					case strings.HasPrefix(msg.Content, "help"):
						// Replace placeholders in help message
						helpMsg := strings.ReplaceAll(stuff.config.Messages.HelpMessage, "<prefix>", prefix)

						// Reply with help
						msg.Reply(context.Background(), stuff.client, helpMsg)

					default:
						// Find the leftmost command
						cmd := spaceSepRegex.FindString(msg.Content)

						// If the user didn't input a command
						if cmd == "" {
							// Subsitute values
							noCmdError := strings.ReplaceAll(stuff.config.Messages.NoCmdError, "<prefix>", prefix)
							// Reply with message
							msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", noCmdError))
						} else { // If the did
							// Subsitute values
							invalidCmdError := strings.ReplaceAll(stuff.config.Messages.InvalidCmdError, "<prefix>", prefix)
							invalidCmdError = strings.ReplaceAll(invalidCmdError, "<command>", cmd)
							// Reply with message
							msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", invalidCmdError))
						}
				}
		}
	}
}
