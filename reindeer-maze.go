package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
)

type dir int

const (
	n dir = iota
	e
	s
	w
)

func parseMsg(msg string) (dir, error) {
	switch msg {
	case "N", "n":
		return n, nil
	case "E", "e":
		return e, nil
	case "S", "s":
		return s, nil
	case "W", "w":
		return w, nil
	}

	return 0, errors.New("invalid message")
}

func client(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	io.WriteString(conn, "Welcome to the reindeer maze! What is your team name?\n")
	if scanner.Scan() {
		name := scanner.Text()
		log.Printf("Team %s joined", name)

		io.WriteString(conn, "You are somewhere\n")
		for scanner.Scan() {
			msg := scanner.Text()
			d, err := parseMsg(msg)
			if err != nil {
				io.WriteString(conn, "Bad command, please try again\n")
				continue
			}

			log.Printf("%s moved %d", name, d)

			io.WriteString(conn, "You are somewhere\n")
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

	log.Printf("Listening on localhost:3000")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go client(conn)
	}
}
