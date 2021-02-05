package main

import "strconv"
import "errors"
import "strings"
import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"
import "go.uber.org/zap"

func getMaze(msg *disgord.Message, prefix string) (maze.Maze, error) {
	// Initalize arguments to defaults
	var width = stuff.config.General.DefaultMazeWidth
	var height = stuff.config.General.DefaultMazeHeight
	var loopy = false

	// idk
	var err error

	// Make a maze to hold the maze
	var coolMaze maze.Maze = &maze.GTreeMaze{}

	// Get arguments from message
	args := strings.Split(msg.Content, " ")

	for i, arg := range args {
		// Checks weather the arg is a size
		isSize := isSizeRegex.MatchString(arg)

		// Checks weather the arg is a type
		isType := isTypeRegex.MatchString(arg)

		switch {
			// If the argument is empty
			case arg == "":
				// Do nothing
				break

			// Too many argunments
			case i >= 3:
				// Substitute values and return error
				tooManyArgsError := strings.ReplaceAll(stuff.config.Messages.TooManyArgsError, "<prefix>", prefix)
				return nil, errors.New(tooManyArgsError)

			// The argument is a size
			case isSize:
				// Seperate width from height
				nums := strings.Split(arg, "x")

				// Set the width and height to non-default
				width, err = strconv.Atoi(nums[0])
				if err != nil {
					stuff.logger.Error("Atoi is broken", zap.String("NUM", nums[0]), zap.String("MSG", msg.Content))
					// Substitute values and return error
					genericError := strings.ReplaceAll(stuff.config.Messages.GenericError, "<prefix>", prefix)
					return nil, errors.New(genericError)
				}
				height, err = strconv.Atoi(nums[1])
				if err != nil {
					stuff.logger.Error("Atoi is broken", zap.String("NUM", nums[1]), zap.String("MSG", msg.Content))
					// Substitute values and return error
					genericError := strings.ReplaceAll(stuff.config.Messages.GenericError, "<prefix>", prefix)
					return nil, errors.New(genericError)
				}

			// The argument is a type
			case isType:
				switch arg {
				case "windy":
					coolMaze = &maze.GTreeMaze{}
				case "spikey":
					coolMaze = &maze.KruskalMaze{}
				case "loopy":
					coolMaze = &maze.GTreeMaze{}
					loopy = true
				}

			// The argument is invalid
			default:
				// Substitute values and return error
				invalidArgError := strings.ReplaceAll(stuff.config.Messages.UnknownArgError, "<prefix>", prefix)
				invalidArgError = strings.ReplaceAll(invalidArgError, "<argument>", arg)
				return nil, errors.New(invalidArgError)
		}
	}

	// Check to make shure that the size is within limits
	if !(width >= 2 && width <= 30 &&
		height >= 2 && height <= 30) {
		return nil, errors.New(stuff.config.Messages.SizeError)
	}

	// Generate a maze
	coolMaze.Generate(width, height)

	// If type loopy make loopy
	if loopy == true {
		coolMaze.Loopify()
	}

	// Set the position to outside the map so The player marker won't display
	coolMaze.SetPos(-1, -1)

	return coolMaze, nil
}