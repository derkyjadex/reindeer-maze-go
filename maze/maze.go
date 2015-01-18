package maze

import (
	"math/rand"
	"time"
)

type Maze struct {
	Width, Height      int
	Walls              [][]bool
	PresentX, PresentY int
	players            map[*Player]bool
	processQueue       chan<- interface{}
}

type addPlayerMsg struct {
	player *Player
}

type removePlayerMsg struct {
	player *Player
}

type movePlayerMsg struct {
	player     *Player
	newX, newY int
	done       chan<- struct{}
}

type getPlayersMsg struct {
	responseChan chan<- []Player
}

func startProcessor(maze *Maze) chan<- interface{} {
	queue := make(chan interface{})
	go func() {
		for msg := range queue {
			switch msg := msg.(type) {
			case addPlayerMsg:
				maze.players[msg.player] = true

			case removePlayerMsg:
				delete(maze.players, msg.player)

			case movePlayerMsg:
				msg.player.X = msg.newX
				msg.player.Y = msg.newY
				msg.done <- struct{}{}

			case getPlayersMsg:
				players := make([]Player, 0, len(maze.players))
				for player, _ := range maze.players {
					players = append(players, *player)
				}

				msg.responseChan <- players
			}
		}
	}()

	return queue
}

func NewMaze(width, height int) *Maze {
	maze := new(Maze)
	maze.Width = width
	maze.Height = height
	maze.players = make(map[*Player]bool)
	maze.PresentX = width / 2
	maze.PresentY = height / 2

	maze.Walls = generateMaze(width, height, maze.PresentX, maze.PresentY)

	maze.processQueue = startProcessor(maze)

	return maze
}

func (maze *Maze) String() string {
	result := ""
	for y := maze.Height - 1; y >= 0; y-- {
		for x := 0; x < maze.Width; x++ {
			if x == maze.PresentX && y == maze.PresentY {
				result += "PP"
			} else if maze.Walls[x][y] {
				result += "██"
			} else {
				result += "  "
			}
		}

		result += "\n"
	}

	return result
}

type point struct {
	x, y int
	d    Dir
}

func generateMaze(width, height, startX, startY int) [][]bool {
	walls := make([][]bool, width)
	for x := 0; x < width; x++ {
		walls[x] = make([]bool, height)
		for y := 0; y < height; y++ {
			walls[x][y] = true
		}
	}

	walls[startX][startY] = false

	wallList := []point{}

	if startX > 0 {
		wallList = append(wallList, point{startX - 1, startY, W})
	}
	if startX < width-1 {
		wallList = append(wallList, point{startX + 1, startY, E})
	}
	if startY > 0 {
		wallList = append(wallList, point{startX, startY - 1, S})
	}
	if startY < height-1 {
		wallList = append(wallList, point{startX, startY + 1, N})
	}

	rand.Seed(time.Now().UnixNano())

	for len(wallList) > 0 {
		i := rand.Intn(len(wallList))
		wall := wallList[i]
		wallList = append(wallList[:i], wallList[i+1:]...)

		x, y := move(wall.x, wall.y, wall.d)
		if x < 0 || x > width-1 || y < 0 || y > height-1 {
			walls[wall.x][wall.y] = false

		} else if walls[wall.x][wall.y] && walls[x][y] {
			walls[wall.x][wall.y] = false
			walls[x][y] = false

			if x > 0 && walls[x-1][y] {
				wallList = append(wallList, point{x - 1, y, W})
			}
			if x < width-1 && walls[x+1][y] {
				wallList = append(wallList, point{x + 1, y, E})
			}
			if y > 0 && walls[x][y-1] {
				wallList = append(wallList, point{x, y - 1, S})
			}
			if y < width-1 && walls[x][y+1] {
				wallList = append(wallList, point{x, y + 1, N})
			}
		}
	}

	return walls
}

func (maze *Maze) Players() []Player {
	results := make(chan []Player)
	maze.processQueue <- getPlayersMsg{results}

	return <-results
}

func (maze *Maze) AddPlayer(name string) *Player {
	rand.Seed(time.Now().UnixNano())

	for {
		x := rand.Intn(maze.Width)
		y := rand.Intn(maze.Height)

		if !maze.Walls[x][y] {
			player := &Player{
				maze: maze,
				Name: name,
				X:    x,
				Y:    y,
			}

			maze.processQueue <- addPlayerMsg{player}

			return player
		}
	}
}

func (maze *Maze) validLocation(x, y int) bool {
	return x >= 0 && x < maze.Width &&
		y >= 0 && y < maze.Height &&
		!maze.Walls[x][y]
}

func (maze *Maze) measureFree(x, y int, d Dir) int {
	for c := 0; ; c++ {
		x, y = move(x, y, d)
		if !maze.validLocation(x, y) {
			return c
		}
	}
}
