module maze-bot

go 1.15

require (
	github.com/andersfylling/disgord v0.23.2
	github.com/iComputeDaily/maze v0.0.0-20201221151754-d45f4b24396a
	github.com/pelletier/go-toml v1.8.1
	github.com/sirupsen/logrus v1.7.0
)

replace (
	github.com/iComputeDaily/maze => ../maze
)
