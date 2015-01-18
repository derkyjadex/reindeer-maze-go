package maze

import (
	"fmt"
)

type Dir int

const (
	N Dir = iota
	E
	S
	W
)

type Compass struct {
	North     int
	East      int
	South     int
	West      int
	Present   *Dir
	OnPresent bool
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
	if compass.OnPresent {
		p = "X"
	} else if compass.Present == nil {
		p = "?"
	} else {
		p = compass.Present.String()
	}

	return fmt.Sprintf("N%d E%d S%d W%d P%s",
		compass.North, compass.East, compass.South, compass.West, p)
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
