package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/derkyjadex/reindeer-maze-go/maze"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func parseMsg(msg string) (maze.Dir, error) {
	switch msg {
	case "N", "n":
		return maze.N, nil
	case "E", "e":
		return maze.E, nil
	case "S", "s":
		return maze.S, nil
	case "W", "w":
		return maze.W, nil
	}

	return 0, errors.New("invalid message")
}

const moveDelay = 100 * time.Millisecond

func client(conn net.Conn, maze *maze.Maze) {
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
			if compass.OnPresent {
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

func show(output io.Writer, walls [][]bool, presentX, presentY int, players []maze.Player) {
	width, height := len(walls), len(walls[0])

	players_ := make(map[string]bool)
	for _, player := range players {
		xy := fmt.Sprintf("%d,%d", player.X, player.Y)
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

func console(maze *maze.Maze) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch scanner.Text() {
		case "maze":
			fmt.Fprint(os.Stderr, maze)

		case "players":
			players := maze.Players()
			fmt.Fprint(os.Stderr, "Players:\n")
			for _, player := range players {
				fmt.Fprintf(os.Stderr, "  %s @ %d, %d\n", player.Name, player.X, player.Y)
			}

		case "show":
			players := maze.Players()
			show(os.Stderr, maze.Walls, maze.PresentX, maze.PresentY, players)
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

func makeMazeHandler(maze *maze.Maze) func(w http.ResponseWriter, r *http.Request) {
	json := fmt.Sprintf("{\"width\":%d,\"height\":%d,\"presentX\":%d,\"presentY\":%d,\"walls\":[",
		maze.Width, maze.Height, maze.PresentX, maze.PresentY)

	for x := 0; x < maze.Width; x++ {
		for y := 0; y < maze.Height; y++ {
			if maze.Walls[x][y] {
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

func makePlayersHandler(maze *maze.Maze) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		json := "[ "
		for _, player := range maze.Players() {
			json += fmt.Sprintf("{\"name\": \"%s\", \"x\": %d, \"y\": %d},", player.Name, player.X, player.Y)
		}

		json = json[:len(json)-1] + "]"

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, json)
	}
}

func webServer(maze *maze.Maze) {
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

	maze := maze.NewMaze(50, 50)

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
