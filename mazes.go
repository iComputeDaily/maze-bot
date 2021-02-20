package main

import "strconv"
import "errors"
import "strings"
import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"
import "go.uber.org/zap"

func parseSizeArg(msg *disgord.Message, prefix string, arg string) (width int, height int, err error) {
	// Seperate width from height
	nums := strings.Split(arg, "x")

	// Set the width
	width, err = strconv.Atoi(nums[0])
	if err != nil {
		stuff.logger.Error("Atoi is broken", zap.String("NUM", nums[0]), zap.String("MSG", msg.Content))

		// Substitute values and return
		err = errors.New(strings.ReplaceAll(stuff.config.Messages.GenericError, "<prefix>", prefix))
		return
	}

	// Set the height
	height, err = strconv.Atoi(nums[1])
	if err != nil {
		stuff.logger.Error("Atoi is broken", zap.String("NUM", nums[0]), zap.String("MSG", msg.Content))

		// Substitute values and return
		err = errors.New(strings.ReplaceAll(stuff.config.Messages.GenericError, "<prefix>", prefix))
		return
	}

	return
}

func getMaze(msg *disgord.Message, prefix string) (maze.Maze, error) {
	// Initalize arguments to defaults
	var (
		width    int       = stuff.config.General.DefaultMazeWidth
		height   int       = stuff.config.General.DefaultMazeHeight
		loopy    bool      = false
		coolMaze maze.Maze = &maze.GTreeMaze{}
	)

	// Get arguments from message
	args := strings.Split(msg.Content, " ")

	for i, arg := range args {
		// Checks weather the arg is a size
		isSize := isSizeRegex.MatchString(arg)

		// Checks weather the arg is a type
		isType := isTypeRegex.MatchString(arg)

		switch {
		case arg == "":
			break

		// Too many argunments
		case i >= 3:
			tooManyArgsError := strings.ReplaceAll(stuff.config.Messages.TooManyArgsError, "<prefix>", prefix)
			return nil, errors.New(tooManyArgsError)

		// The argument is a size
		case isSize:
			// Avoid redecloration of width causing syntax error
			var err error

			width, height, err = parseSizeArg(msg, prefix, arg)
			if err != nil {
				return nil, err
			}

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

	// Set the position to outside the map so the player marker won't display
	coolMaze.SetPos(-1, -1)

	return coolMaze, nil
}
