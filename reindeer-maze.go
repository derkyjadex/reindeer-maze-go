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
	default:
		return ""
	}
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

	return fmt.Sprintf("N%d S%d E%d W%d P%s",
		compass.north, compass.south, compass.east, compass.west, p)
}

func NewMaze(width, height int) *Maze {
	maze := new(Maze)
	maze.width = width
	maze.height = height

	maze.walls = make([][]bool, width)
	for i := 0; i < width; i++ {
		maze.walls[i] = make([]bool, height)
	}

	maze.players = make(map[*Player]bool)
	maze.presentX = width / 2
	maze.presentY = height / 2

	return maze
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
		} else {
			present = S
		}

		presentPtr = &present

	} else if y == maze.presentY {
		if x < maze.presentX {
			present = E
		} else {
			present = W
		}

		presentPtr = &present
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

		log.Printf("Team %s joined", name)

		io.WriteString(conn, player.Compass().String()+"\n")
		for scanner.Scan() {
			msg := scanner.Text()
			d, err := parseMsg(msg)
			if err != nil {
				io.WriteString(conn, "Bad command, please try again\n")
				continue
			}

			if player.Move(d) {
				log.Printf("%s moved %v", name, d)
			}

			io.WriteString(conn, player.Compass().String()+"\n")
		}

		log.Printf("Team %s disconnected", name)
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
