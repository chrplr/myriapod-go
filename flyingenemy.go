package main

import (
	"math/rand"
	"strconv"
)

// FlyingEnemy ("meanie") crosses the play area horizontally while bobbing up and
// down, occasionally pausing its horizontal motion.
type FlyingEnemy struct {
	Sprite
	movingX int // 1 = moving horizontally too, 0 = only vertically
	dx, dy  int
	typ     int
	health  int
	timer   int
}

func NewFlyingEnemy(playerX float64) *FlyingEnemy {
	// Choose the starting side, avoiding a spawn right on top of the player.
	var side int
	switch {
	case playerX < 160:
		side = 1
	case playerX > 320:
		side = 0
	default:
		side = rand.Intn(2)
	}

	return &FlyingEnemy{
		Sprite:  newSprite("blank", float64(550*side-35), 688),
		movingX: 1,
		dx:      1 - 2*side,
		dy:      []int{-1, 1}[rand.Intn(2)],
		typ:     rand.Intn(3),
		health:  1,
	}
}

func (f *FlyingEnemy) Update(g *Game) {
	f.timer++

	f.X += float64(f.dx * f.movingX * (3 - abs(f.dy)))
	f.Y += float64(f.dy * (3 - abs(f.dx*f.movingX)))

	if f.Y < 592 || f.Y > 784 {
		// Gone too high or low - reverse vertical direction.
		f.movingX = rand.Intn(2)
		f.dy = -f.dy
	}

	animFrame := []int{0, 2, 1, 2}[(f.timer/4)%4]
	f.Image = "meanie" + strconv.Itoa(f.typ) + strconv.Itoa(animFrame)
}
