package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
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
	presentX, presentY int
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
				msg.player.x = msg.newX
				msg.player.y = msg.newY
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
	maze.width = width
	maze.height = height
	maze.players = make(map[*Player]bool)
	maze.presentX = width / 2
	maze.presentY = height / 2

	maze.walls = generateMaze(width, height, maze.presentX, maze.presentY)

	maze.processQueue = startProcessor(maze)

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

func (maze *Maze) Players() []Player {
	results := make(chan []Player)
	maze.processQueue <- getPlayersMsg{results}

	return <-results
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

			maze.processQueue <- addPlayerMsg{player}

			return player
		}
	}
}

func (player *Player) Remove() {
	player.maze.processQueue <- removePlayerMsg{player}
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
		done := make(chan struct{})
		player.maze.processQueue <- movePlayerMsg{player, x, y, done}
		<-done

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

const moveDelay = 100 * time.Millisecond

func client(conn net.Conn, maze *Maze) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	io.WriteString(conn, "Welcome to the reindeer maze! What is your team name?\n")
	if scanner.Scan() {
		name := scanner.Text()

		player := maze.AddPlayer(name)
		defer player.Remove()

		log.Printf("%s joined", name)

		moveStart := time.Now()

		io.WriteString(conn, player.Compass().String()+"\n")
		for scanner.Scan() {
			time.Sleep(moveDelay - time.Since(moveStart))
			moveStart = time.Now()

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

func show(output io.Writer, walls [][]bool, presentX, presentY int, players []Player) {
	width, height := len(walls), len(walls[0])

	players_ := make(map[string]bool)
	for _, player := range players {
		xy := fmt.Sprintf("%d,%d", player.x, player.y)
		players_[xy] = true
	}

	fmt.Fprint(output, "\033(0")

	for y := height - 1; y >= 0; y -= 1 {
		for x := 0; x < width; x += 1 {
			if players_[fmt.Sprintf("%d,%d", x, y)] {
				fmt.Fprint(output, "aa")

			} else if walls[x][y] {
				sides := ""
				if y == height-1 || walls[x][y+1] {
					sides += "n"
				}
				if x == width-1 || walls[x+1][y] {
					sides += "e"
				}
				if y == 0 || walls[x][y-1] {
					sides += "s"
				}
				if x == 0 || walls[x-1][y] {
					sides += "w"
				}

				switch sides {
				case "":
					fmt.Fprint(output, "~")
				case "nesw":
					fmt.Fprint(output, "nn")
				case "n":
					fmt.Fprint(output, "mj")
				case "s":
					fmt.Fprint(output, "lk")
				case "ns":
					fmt.Fprint(output, "xx")
				case "w", "e", "ew":
					fmt.Fprint(output, "qq")
				case "ne":
					fmt.Fprint(output, "mv")
				case "es":
					fmt.Fprint(output, "lw")
				case "sw":
					fmt.Fprint(output, "wk")
				case "nw":
					fmt.Fprint(output, "vj")
				case "nes":
					fmt.Fprint(output, "tn")
				case "esw":
					fmt.Fprint(output, "ww")
				case "nsw":
					fmt.Fprint(output, "nu")
				case "new":
					fmt.Fprint(output, "vv")
				}
			} else if x == presentX && y == presentY {
				fmt.Fprint(output, "``")
			} else {
				fmt.Fprint(output, "  ")
			}
		}

		fmt.Fprint(output, "\n")
	}

	fmt.Fprint(output, "\033(B")
}

func console(maze *Maze) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch scanner.Text() {
		case "maze":
			fmt.Fprint(os.Stderr, maze)

		case "players":
			players := maze.Players()
			fmt.Fprint(os.Stderr, "Players:\n")
			for _, player := range players {
				fmt.Fprintf(os.Stderr, "  %s @ %d, %d\n", player.name, player.x, player.y)
			}

		case "show":
			players := maze.Players()
			show(os.Stderr, maze.walls, maze.presentX, maze.presentY, players)
		}
	}
}

func makeFileHandler(path, contentType string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("Error serving '%s': %v", path, err)
			http.Error(w, ":(", 500)
		}

		w.Write(content)
	}
}

func makeMazeHandler(maze *Maze) func(w http.ResponseWriter, r *http.Request) {
	json := fmt.Sprintf("{\"width\":%d,\"height\":%d,\"presentX\":%d,\"presentY\":%d,\"walls\":[",
		maze.width, maze.height, maze.presentX, maze.presentY)

	for x := 0; x < maze.width; x++ {
		for y := 0; y < maze.height; y++ {
			if maze.walls[x][y] {
				json += fmt.Sprintf("[%d,%d],", x, y)
			}
		}
	}

	json = json[:len(json)-1] + "]}"

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, json)
	}
}

func makePlayersHandler(maze *Maze) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		json := "[ "
		for _, player := range maze.Players() {
			json += fmt.Sprintf("{\"name\": \"%s\", \"x\": %d, \"y\": %d},", player.name, player.x, player.y)
		}

		json = json[:len(json)-1] + "]"

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, json)
	}
}

func webServer(maze *Maze) {
	http.HandleFunc("/", makeFileHandler("index.html", "text/html"))
	http.HandleFunc("/reindeer.js", makeFileHandler("reindeer.js", "application/javascript"))
	http.HandleFunc("/maze", makeMazeHandler(maze))
	http.HandleFunc("/players", makePlayersHandler(maze))
	http.ListenAndServe("localhost:3001", nil)
}

func main() {
	log.Printf("Starting up...")

	l, err := net.Listen("tcp", "localhost:3000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	maze := NewMaze(50, 50)

	log.Printf("Listening on localhost:3000")

	go console(maze)
	go webServer(maze)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go client(conn, maze)
	}
}
