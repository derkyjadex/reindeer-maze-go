package maze

type Player struct {
	maze *Maze
	Name string
	X, Y int
}

func (player *Player) Remove() {
	player.maze.processQueue <- removePlayerMsg{player}
}

func (player *Player) Compass() Compass {
	maze := player.maze
	x, y := player.X, player.Y

	var present Dir
	var presentPtr *Dir
	var onPresent bool

	if x == maze.PresentX && y == maze.PresentY {
		onPresent = true

	} else if x == maze.PresentX {
		if y < maze.PresentY {
			present = N
			if maze.measureFree(x, y, N) >= maze.PresentY-y {
				presentPtr = &present
			}
		} else {
			present = S
			if maze.measureFree(x, y, S) >= y-maze.PresentY {
				presentPtr = &present
			}
		}

	} else if y == maze.PresentY {
		if x < maze.PresentX {
			present = E
			if maze.measureFree(x, y, E) >= maze.PresentX-x {
				presentPtr = &present
			}
		} else {
			present = W
			if maze.measureFree(x, y, W) >= x-maze.PresentX {
				presentPtr = &present
			}
		}
	}

	return Compass{
		North:     maze.measureFree(x, y, N),
		East:      maze.measureFree(x, y, E),
		South:     maze.measureFree(x, y, S),
		West:      maze.measureFree(x, y, W),
		Present:   presentPtr,
		OnPresent: onPresent,
	}
}

func (player *Player) Move(d Dir) bool {
	x, y := move(player.X, player.Y, d)

	if player.maze.validLocation(x, y) {
		done := make(chan struct{})
		player.maze.processQueue <- movePlayerMsg{player, x, y, done}
		<-done

		return true

	} else {
		return false
	}
}
