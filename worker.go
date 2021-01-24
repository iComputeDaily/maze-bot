package main

import "github.com/andersfylling/disgord"
import "context"
import "fmt"

// Listens for messges in the channel and deals with them
func (stuff *stuff) worker(
	helpEvtChan <-chan *disgord.MessageCreate,
	genEvtChan <-chan *disgord.MessageCreate,
	invalidEventChan <-chan *disgord.MessageCreate) {
	
	var helpMsg = stuff.config.General.HelpMessage
	
	for {
		var msg *disgord.Message
		
		// Wait for messages
		select {
			case data, ok := <-helpEvtChan:
				if !ok {
					stuff.logger.Panic("Help channel is dead!")
					return
				}
				msg = data.Message
				
				// Reply with help
				msg.Reply(context.Background(), stuff.client, helpMsg)
				
			case data, ok := <-genEvtChan:
				if !ok {
					stuff.logger.Panic("Gen channel is dead!")
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
				}

			case data, ok := <-invalidEventChan:
				if !ok {
					stuff.logger.Panic("Invalid channel is dead!")
					return
				}
				msg = data.Message

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