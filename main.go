package main

import "fmt"
import "math/rand"
import "time"
import "github.com/iComputeDaily/maze"

// Represents confic options related to discord
type discord struct {
	ProjectName string
	BotToken string
	StatusMessage string
}

// Represents the config file
type config struct {
	Discord discord
}

func initalize() {
	// Initalize the random number generator
	rand.Seed(time.Now().UTC().UnixNano())
	
	return
}

func main() {
	initalize()
	
	var maze maze.Maze = &maze.KruskalMaze{}
	
	maze.Generate(20, 20)
	
	stringyMaze := maze.Stringify()
	
	fmt.Println(stringyMaze)
}
