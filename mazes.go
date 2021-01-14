package main

import "strconv"
import "fmt"
import "errors"
import "regexp"
import "strings"
import "github.com/iComputeDaily/maze"
import "github.com/andersfylling/disgord"

func (stuff *stuff) getMaze(msg *disgord.Message) (maze.Maze, error) {
	// Initalize arguments to defaults
	var width = stuff.config.General.DefaultMazeWidth
	var height = stuff.config.General.DefaultMazeHeight

	// Get arguments from message
	args := strings.Split(msg.Content, " ")

	// Debugging
	fmt.Println("args:", args)

	for i, arg := range args {
		// Checks weather the arg is a size
		isSize, err := regexp.MatchString(`^(?i)\d+x\d+$`, arg)
		// Make shure the regrex didn't break
		if err != nil {
			return nil, errors.New("Regrex didn't work; This is probobly a bug.")
		}

		// Checks weather the arg is a type
		// isType, err := regexp.MatchString(`^(?i)spikey|windy|loopy$`, arg)
		// Make shure the regrex didn't break
		// if err != nil {
		// 	return nil, errors.New("Regrex didn't work; This is probobly a bug.")
		// }

		switch {
			// If the argument is empty
			case arg == "":
				// Do nothing
				break

			// Too many argunments
			case i >= 2:
				return nil, errors.New("Too many arguments!")

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
			// case isType:

			// The argument is invaled
			default:
				return nil, errors.New(fmt.Sprintln("Unknown argument `", arg, "`.\nRun `!maze help` for usage."))
		}
	}

	// Generate a maze
	var coolMaze maze.Maze = &maze.GTreeMaze{}
	coolMaze.Generate(width, height)

	return coolMaze, nil
}