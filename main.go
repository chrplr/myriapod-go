package main

import (
	"flag"
	"strconv"

	"github.com/chrplr/pgzgo"
)

const (
	Width  = 480
	Height = 800
)

// --- small numeric helpers ---

func abs[T int | float64](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// pmod is Python-style modulo: the result has the sign of the divisor.
func pmod(a, b int) int {
	m := a % b
	if m < 0 {
		m += b
	}
	return m
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// b converts a bool to 0/1; b2s to "0"/"1".
func b(v bool) int {
	if v {
		return 1
	}
	return 0
}

func b2s(v bool) string { return strconv.Itoa(b(v)) }

// --- game state ---

type State int

const (
	StateMenu State = iota
	StatePlay
	StateGameOver
)

var (
	state     State
	game      *Game
	spaceDown bool

	assets *Assets
	audio  *Audio
)

// spacePressed reports a fresh press of a start key — Space or Enter — (down
// now, up last frame).
func spacePressed() bool {
	if keyDown_space() || keyDown_enter() {
		if spaceDown {
			return false
		}
		spaceDown = true
		return true
	}
	spaceDown = false
	return false
}

func update() {
	switch state {
	case StateMenu:
		if spacePressed() {
			state = StatePlay
			game = NewGame(NewPlayer(240, 768), assets, audio)
		}
		game.Update()

	case StatePlay:
		if game.player.lives == 0 && game.player.timer == 100 {
			audio.Play("gameover")
			state = StateGameOver
		} else {
			game.Update()
		}

	case StateGameOver:
		if spacePressed() {
			state = StateMenu
			game = NewGame(nil, assets, audio)
		}
	}
}

func draw() {
	game.Draw()

	switch state {
	case StateMenu:
		assets.Blit("title", 0, 0)
		assets.Blit("space"+strconv.Itoa((game.time/4)%14), 0, 420)

	case StatePlay:
		for i := 0; i < game.player.lives; i++ {
			assets.Blit("life", float64(i*40+8), 4)
		}
		score := strconv.Itoa(game.score)
		for i := 1; i <= len(score); i++ {
			digit := string(score[len(score)-i])
			assets.Blit("digit"+digit, float64(468-i*24), 5)
		}

	case StateGameOver:
		assets.Blit("over", 0, 0)
	}
}

func main() {
	flag.Parse()

	a, err := pgzgo.New(pgzgo.Config{
		Title:  "Myriapod",
		Width:  Width,
		Height: Height,
		Images: imagesFS,
		Audio:  audioFS,
	})
	if err != nil {
		panic(err)
	}
	defer a.Close()

	app = a
	assets = a.Screen
	audio = a.Audio

	// The original started the looping theme as soon as the mixer was ready.
	audio.PlayMusic("theme", 0.4)

	state = StateMenu
	// The initial game has no player: it runs as the attract-mode demo.
	game = NewGame(nil, assets, audio)

	a.Loop(
		func(*pgzgo.App) { update() },
		func(*pgzgo.App) { draw() },
	)
}
