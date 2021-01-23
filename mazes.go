package main

import "strconv"
import "fmt"
import "errors"
import "strings"
import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"

func (stuff *stuff) getMaze(msg *disgord.Message) (maze.Maze, error) {
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
				return nil, errors.New("Too many arguments. Run `!maze help` for usage help.")

			// The argument is a size
			case isSize:
				// Seperate width from height
				nums := strings.Split(arg, "x")

				// Set the width and height to non-default
				width, err = strconv.Atoi(nums[0])
				if err != nil {
					return nil, errors.New("Atoi didn't work; This is probobly a bug.")
				}
				height, err = strconv.Atoi(nums[1])
				if err != nil {
					return nil, errors.New("Atoi didn't work; This is probobly a bug.")
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

			// The argument is invaled
			default:
				return nil, errors.New(fmt.Sprintln("Unknown argument `", arg, "`. Run `!maze help` for usage help."))
		}
	}

	// Check to make shure that the size is within limits
	if !(width >= 2 && width <= 30 &&
		height >= 2 && height <= 30) {
		return nil, errors.New("Size must be at least `2x2`, and at most `30x30`(due to discords charachter limit)")
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