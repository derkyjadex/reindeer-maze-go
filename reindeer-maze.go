package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

type Dir int

const (
	N Dir = iota
	E
	S
	W
)

type Player struct {
	maze *Maze
	name string
	x, y int
}

type Maze struct {
	width, height      int
	walls              [][]bool
	players            map[*Player]bool
	presentX, presentY int
}

type Compass struct {
	north     int
	east      int
	south     int
	west      int
	present   *Dir
	onPresent bool
}

func (d Dir) String() string {
	switch d {
	case N:
		return "N"
	case E:
		return "E"
	case S:
		return "S"
	case W:
		return "W"
	}

	panic("Invalid direction")
}

func (compass Compass) String() string {
	var p string
	if compass.onPresent {
		p = "X"
	} else if compass.present == nil {
		p = "?"
	} else {
		p = compass.present.String()
	}

	return fmt.Sprintf("N%d E%d S%d W%d P%s",
		compass.north, compass.east, compass.south, compass.west, p)
}

func NewMaze(width, height int) *Maze {
	maze := new(Maze)
	maze.width = width
	maze.height = height
	maze.players = make(map[*Player]bool)
	maze.presentX = width / 2
	maze.presentY = height / 2

	maze.walls = generateMaze(width, height, maze.presentX, maze.presentY)

	return maze
}

func (maze *Maze) String() string {
	result := ""
	for y := maze.height - 1; y >= 0; y-- {
		for x := 0; x < maze.width; x++ {
			if x == maze.presentX && y == maze.presentY {
				result += "PP"
			} else if maze.walls[x][y] {
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

func (maze *Maze) AddPlayer(name string) *Player {
	rand.Seed(time.Now().UnixNano())

	for {
		x := rand.Intn(maze.width)
		y := rand.Intn(maze.height)

		if !maze.walls[x][y] {
			player := &Player{
				maze: maze,
				name: name,
				x:    x,
				y:    y,
			}
			maze.players[player] = true

			return player
		}
	}
}

func (player *Player) Remove() {
	delete(player.maze.players, player)
}

func move(x, y int, d Dir) (int, int) {
	switch d {
	case N:
		y++
	case E:
		x++
	case S:
		y--
	case W:
		x--
	}

	return x, y
}

func (maze *Maze) validLocation(x, y int) bool {
	return x >= 0 && x < maze.width &&
		y >= 0 && y < maze.height &&
		!maze.walls[x][y]
}

func (maze *Maze) measureFree(x, y int, d Dir) int {
	for c := 0; ; c++ {
		x, y = move(x, y, d)
		if !maze.validLocation(x, y) {
			return c
		}
	}
}

func (player *Player) Compass() Compass {
	maze := player.maze
	x, y := player.x, player.y

	var present Dir
	var presentPtr *Dir
	var onPresent bool

	if x == maze.presentX && y == maze.presentY {
		onPresent = true

	} else if x == maze.presentX {
		if y < maze.presentY {
			present = N
			if maze.measureFree(x, y, N) >= maze.presentY-y {
				presentPtr = &present
			}
		} else {
			present = S
			if maze.measureFree(x, y, S) >= y-maze.presentY {
				presentPtr = &present
			}
		}

	} else if y == maze.presentY {
		if x < maze.presentX {
			present = E
			if maze.measureFree(x, y, E) >= maze.presentX-x {
				presentPtr = &present
			}
		} else {
			present = W
			if maze.measureFree(x, y, W) >= x-maze.presentX {
				presentPtr = &present
			}
		}
	}

	return Compass{
		north:     maze.measureFree(x, y, N),
		east:      maze.measureFree(x, y, E),
		south:     maze.measureFree(x, y, S),
		west:      maze.measureFree(x, y, W),
		present:   presentPtr,
		onPresent: onPresent,
	}
}

func (player *Player) Move(d Dir) bool {
	x, y := move(player.x, player.y, d)

	if player.maze.validLocation(x, y) {
		player.x = x
		player.y = y

		return true

	} else {
		return false
	}
}

func parseMsg(msg string) (Dir, error) {
	switch msg {
	case "N", "n":
		return N, nil
	case "E", "e":
		return E, nil
	case "S", "s":
		return S, nil
	case "W", "w":
		return W, nil
	}

	return 0, errors.New("invalid message")
}

func client(conn net.Conn, maze *Maze) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	io.WriteString(conn, "Welcome to the reindeer maze! What is your team name?\n")
	if scanner.Scan() {
		name := scanner.Text()

		player := maze.AddPlayer(name)
		defer player.Remove()

		log.Printf("%s joined", name)

		io.WriteString(conn, player.Compass().String()+"\n")
		for scanner.Scan() {
			msg := scanner.Text()
			d, err := parseMsg(msg)
			if err != nil {
				io.WriteString(conn, "Bad command, please try again\n")
				continue
			}

			player.Move(d)

			compass := player.Compass()
			if compass.onPresent {
				log.Printf("%s found the present", name)
			}

			io.WriteString(conn, compass.String()+"\n")
		}

		log.Printf("%s disconnected", name)
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func main() {
	log.Printf("Starting up...")

	l, err := net.Listen("tcp", "localhost:3000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	maze := NewMaze(100, 100)

	log.Printf("Listening on localhost:3000")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go client(conn, maze)
	}
}
