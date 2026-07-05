package main

import "strconv"

// Explosion is a short animation. Type 0/1/2 selects the sprite set; the frame
// advances every four frames. The Game removes it once timer reaches 31.
type Explosion struct {
	Sprite
	typ   int
	timer int
}

func NewExplosion(x, y float64, typ int) *Explosion {
	return &Explosion{Sprite: newSprite("blank", x, y), typ: typ}
}

func (e *Explosion) Update(g *Game) {
	e.timer++
	e.Image = "exp" + strconv.Itoa(e.typ) + strconv.Itoa(e.timer/4)
}
