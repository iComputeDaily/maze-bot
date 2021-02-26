package main

import "strconv"
import "errors"
import "strings"
import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"
import "go.uber.org/zap"

func parseNum(strNum string, prefix string, logger *zap.SugaredLogger) (num int, err error) {
	// Make shure size isn't so big it will break atoi
	if !(len(strNum) <= 3) {
		logger.Infow("Size is invalid", "Num", strNum)
		return 0, errors.New(stuff.config.Messages.SizeError)
	}

	// Set the size
	num, err = strconv.Atoi(strNum)

	if err != nil {
		logger.Errorw("Atoi is broken",
			"Num", strNum,
			"error", err,
		)

		// Substitute values and return
		err = errors.New(strings.ReplaceAll(stuff.config.Messages.GenericError, "<prefix>", prefix))
		return
	}

	// Check to make shure that the size is within limits
	if !(num >= 2 && num <= 30) {
		logger.Infow("Size is invalid", "Num", num)
		return 0, errors.New(stuff.config.Messages.SizeError)
	}

	return
}

func parseSizeArg(prefix string, arg string, logger *zap.SugaredLogger) (width int, height int, err error) {
	// Seperate width from height
	nums := strings.Split(arg, "x")

	// Set the width
	width, err = parseNum(nums[0], prefix, logger)
	if err != nil {
		return 0, 0, err
	}

	// Set the height
	height, err = parseNum(nums[1], prefix, logger)
	if err != nil {
		return 0, 0, err
	}

	return
}

func getMaze(msg *disgord.Message, prefix string, logger *zap.SugaredLogger) (maze.Maze, *zap.SugaredLogger, error) {
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
			logger.Infow("Command has too many arguments")

			tooManyArgsError := strings.ReplaceAll(stuff.config.Messages.TooManyArgsError, "<prefix>", prefix)
			return nil, logger, errors.New(tooManyArgsError)

		// The argument is a size
		case isSize:
			// Avoid redecloration of width causing syntax error
			var err error

			width, height, err = parseSizeArg(prefix, arg, logger)
			if err != nil {
				return nil, logger, err
			}

			// Add size logging context
			logger = logger.With(
				"Width", width,
				"Height", height,
			)

		case isType:
			switch arg {
			case "windy":
				logger = logger.With("MazeType", "windy")
				coolMaze = &maze.GTreeMaze{}
			case "spikey":
				logger = logger.With("MazeType", "spikey")
				coolMaze = &maze.KruskalMaze{}
			case "loopy":
				logger = logger.With("MazeType", "loopy")
				coolMaze = &maze.GTreeMaze{}
				loopy = true
			}

		// The argument is invalid
		default:
			logger.Infow("Command has invalid argument", "Argument", arg)

			invalidArgError := strings.ReplaceAll(stuff.config.Messages.UnknownArgError, "<prefix>", prefix)
			invalidArgError = strings.ReplaceAll(invalidArgError, "<argument>", arg)
			return nil, logger, errors.New(invalidArgError)
		}
	}

	// Generate a maze
	coolMaze.Generate(width, height)

	// If type loopy make loopy
	if loopy == true {
		coolMaze.Loopify()
	}

	// Set the position to outside the map so the player marker won't display
	coolMaze.SetPos(-1, -1)

	return coolMaze, logger, nil
}
