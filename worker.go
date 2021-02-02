package main

import "github.com/andersfylling/disgord"
import "context"
import "fmt"
import "strings"

// Listens for messges in the channel and deals with them
func (stuff *stuff) worker(msgEvtChan <-chan *disgord.MessageCreate) {
	
	var helpMsg = strings.ReplaceAll(stuff.config.General.HelpMessage, "<prefix>", "!")
	
	for {
		var msg *disgord.Message
		
		// Wait for messages
		select {
			case data, ok := <-msgEvtChan:
				if !ok {
					stuff.logger.Panic("Invalid channel is dead!")
					return
				}
				msg = data.Message

				msg.Content = strings.TrimSpace(msg.Content)

				switch {
					case strings.HasPrefix(msg.Content, "gen"):
						msg.Content = strings.TrimPrefix(msg.Content, "gen")

						// Get the maze from the message
						coolMaze, err := stuff.getMaze(msg)
						if err != nil {
							msg.Reply(context.Background(), stuff.client, fmt.Sprintln("Error:", err))

						} else {
							// Reply to the message with the maze
							msg.Reply(context.Background(), stuff.client, "```maze\n" + coolMaze.Stringify() + "```")
						}

					case strings.HasPrefix(msg.Content, "help"):
						// Reply with help
						msg.Reply(context.Background(), stuff.client, helpMsg)

					default:
						// Find the leftmost command
						cmd := spaceSepRegex.FindString(msg.Content)

						// If the user didn't input a command
						if cmd == "" {
							msg.Reply(context.Background(), stuff.client,
							"Error: No command provided. Use `!maze help` for usage help.")
						} else { // If the did
							msg.Reply(context.Background(), stuff.client,
							fmt.Sprintln("Error: Invalid command `", cmd, "`. Use `!maze help` for usage help."))
						}
				}
		}
	}
}