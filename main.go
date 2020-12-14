package main

import "fmt"
import "math/rand"
import "time"
import "io/ioutil"
import toml "github.com/pelletier/go-toml"
import "github.com/iComputeDaily/maze"

// Represents confic options related to discord
type Discord struct {
	ProjectName string
	BotToken string
	StatusMessage string
	Prefix string
}

// Represents the config file
type Config struct {
	Discord Discord
}

func initalize() {
	// Initalize the random number generator
	rand.Seed(time.Now().UTC().UnixNano())
	
	// Read the config file
	file, err := ioutil.ReadFile("config.toml")
	if err != nil {
		fmt.Println("Failed to read config file: ", err)
	}
	
	// Parse the data
	config := Config{}
	err = toml.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Failed to read parse file: ", err)
	}
	
	fmt.Printf("%+v\n", config)
	
	return
}

func main() {
	initalize()
	
	var maze maze.Maze = &maze.KruskalMaze{}
	
	maze.Generate(20, 20)
	
	stringyMaze := maze.Stringify()
	
	fmt.Println(stringyMaze)
}
