package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"time"

	"golang.org/x/term"
)

const SHIFT = 2

const COLS = 30
const ROWS = 15

const PLAYER_HEAD = "█"
const PLAYER_BODY = "▒"
const FOOD = "0"

// ─┌┐│└┘
func renderBoundaries() {
	print("\x1B[?25l\x1B[2J\x1B[0;0H")
	for i := range ROWS + SHIFT {
		if i == 0 || i == ROWS+1 {
			if i == 0 {
				print("┌")
			} else {
				print("└")
			}
			for range COLS {
				print("─")
			}
			if i == 0 {
				print("┐")
			} else {
				print("┘")
			}
		} else {
			print("│")
			for range COLS {
				print(" ")
			}
			print("│")
		}
		print("\x1B[1E")
	}
}

type Event struct {
	Key rune
}

func toScreen(p ...int) []int {
	return []int{p[0] + SHIFT, p[1] + SHIFT}
}

type Pos []int

type Player struct {
	body []Pos
	dir  []int
	grow bool
}

func (player *Player) Update() {
	head := player.body[0]
	x := head[0] + player.dir[0]
	y := head[1] + player.dir[1]

	if y < 0 {
		y = ROWS - 1
	}

	if x < 0 {
		x = COLS - 1
	}

	x = max(0, min(COLS, x%COLS))
	y = max(0, min(ROWS, y%ROWS))

	if len(player.body) > 0 {
		player.body = player.body[:len(player.body)-1]
		player.body = append([]Pos{{head[0], head[1]}}, player.body...)
	}

	player.body[0][0] = x
	player.body[0][1] = y
}

func (player *Player) Grow() {
	size := len(player.body)
	player.body = append(player.body, Pos{
		player.body[0][0] + player.dir[0]*size*-1,
		player.body[0][1] + player.dir[1]*size*-1,
	})
}

func (player *Player) Render() {
	for i, pos := range player.body {
		pos = toScreen(pos[0], pos[1])
		cell := PLAYER_HEAD
		if i > 0 {
			cell = PLAYER_BODY
		}
		fmt.Printf("\x1B[%d;%dH%s", pos[1], pos[0], cell)
	}
}

type Food struct {
	pos []int
}

func (food *Food) Render() {
	pos := toScreen(food.pos[0], food.pos[1])
	fmt.Printf("\x1B[%d;%dH%s", pos[1], pos[0], FOOD)
}

func randomPos() []int {
	return []int{
		rand.Intn(COLS - 1),
		rand.Intn(ROWS - 1),
	}
}

type Game struct {
	player *Player
	food   *Food
	score  int
	over   bool
}

func NewGame() *Game {
	player := &Player{
		body: []Pos{randomPos()},
		dir:  []int{0, 0},
		grow: false,
	}

	food := &Food{
		pos: []int{-1, -1},
	}

	game := &Game{
		player: player,
		food:   food,
		score:  0,
		over:   false,
	}

	game.NewFood()

	return game
}

func (game *Game) Render() {
	renderBoundaries()

	if game.over {
		text := "GAME OVER"
		x := COLS/2 - len(text)/2 - 1
		y := (ROWS / 2) + SHIFT

		info := "[r]estart [q]uit"

		fmt.Printf("\x1B[%d;%dH%s", y-1, x, "╔═══════════╗")
		fmt.Printf("\x1B[%d;%dH║ %s ║", y, x, text)
		fmt.Printf("\x1B[%d;%dH%s", y+1, x, "╚═══════════╝")
		fmt.Printf("\x1B[%d;%dH%s", y+2, len(info)/2, info)
	} else {
		game.player.Render()
		game.food.Render()
		fmt.Printf("\x1B[%d;%dHScore: %d", toScreen(0, ROWS+1)[1], 0, game.score)
	}

}

func (game *Game) NewFood() {
	for {
		pos := randomPos()
		for _, p := range game.player.body {
			if pos[0] != p[0] && pos[1] != p[1] {
				game.food.pos = pos
				return
			}
		}
	}
}

func (game *Game) Update() {
	head := game.player.body[0]

	for _, pos := range game.player.body[1:] {
		if head[0] == pos[0] && head[1] == pos[1] {
			game.over = true
			return
		}
	}

	if head[0] == game.food.pos[0] && head[1] == game.food.pos[1] {
		game.Eat()
	}

	game.player.Update()
}

func (game *Game) Eat() {
	game.NewFood()
	game.player.Grow()
	game.score++
}

func (game *Game) Move(x int, y int) {
	game.player.dir[0] = x
	game.player.dir[1] = y
}

func main() {
	fd := int(os.Stdin.Fd())

	old, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Println("Unable to turn raw mode on")
		os.Exit(1)
	}
	defer term.Restore(fd, old)

	input := make(chan Event)
	quit := make(chan struct{})

	go func() {
		for {
			ev := readInput()
			if ev.Key == 'q' {
				close(quit)
				print("\x1B[?25h")
				return
			}

			input <- ev
		}
	}()

	ticker := time.NewTicker(time.Second / 12)
	defer ticker.Stop()

	game := NewGame()

	for {
		select {
		case <-ticker.C:
			game.Update()
			game.Render()
		case ev := <-input:
			if ev.Key == 'k' || ev.Key == 'w' {
				game.Move(0, -1)
			}

			if ev.Key == 'j' || ev.Key == 's' {
				game.Move(0, 1)
			}

			if ev.Key == 'l' || ev.Key == 'd' {
				game.Move(1, 0)
			}

			if ev.Key == 'h' || ev.Key == 'a' {
				game.Move(-1, 0)
			}

			if game.over && ev.Key == 'r' {
				game = NewGame()
				game.over = false
			}
		case <-quit:
			return
		}
	}
}

var reader = bufio.NewReader(os.Stdin)

func readInput() Event {
	key, _, err := reader.ReadRune()
	if err != nil {
		return Event{
			Key: 'q',
		}
	}
	return Event{Key: key}
}
