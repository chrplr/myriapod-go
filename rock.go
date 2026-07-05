package main

import (
	"math/rand"
	"strconv"
)

// Rock occupies a grid cell. Normal rocks take 3-4 hits; "totem" rocks (health 5)
// award bonus points and revert to a normal rock if not destroyed in time.
type Rock struct {
	Sprite
	typ        int
	health     int
	showHealth int
	timer      int
}

func NewRock(g *Game, x, y int, totem bool) *Rock {
	px, py := cell2pos(x, y, 0, 0)
	r := &Rock{
		Sprite: newSprite("blank", px, py),
		typ:    rand.Intn(4),
		timer:  1,
	}
	if totem {
		// Totem rocks are taller, so pin a custom anchor at their base.
		r.anchorCentre = false
		r.ax, r.ay = 24, 60
		g.PlaySound("totem_create", 1)
		r.health = 5
		r.showHealth = 5
	} else {
		// Normal rocks start looking like they have one health and "grow".
		r.health = rand.Intn(2) + 3
		r.showHealth = 1
	}
	return r
}

// Damage applies damage and returns true if the rock was destroyed.
func (r *Rock) Damage(g *Game, amount int, byBullet bool) bool {
	if byBullet && r.health == 5 {
		g.PlaySound("totem_destroy", 1)
		g.score += 100
	} else if amount > r.health-1 {
		g.PlaySound("rock_destroy", 1)
	} else {
		g.PlaySound("hit", 4)
	}

	expType := 0
	if r.health == 5 {
		expType = 2
	}
	g.explosions = append(g.explosions, NewExplosion(r.X, r.Y, expType))

	r.health -= amount
	r.showHealth = r.health

	// Reverting to the centre anchor keeps the rock in place as it shrinks.
	r.anchorCentre = true

	return r.health < 1
}

func (r *Rock) Update(g *Game) {
	r.timer++

	// Every other frame, advance the growing animation.
	if r.timer%2 == 1 && r.showHealth < r.health {
		r.showHealth++
	}

	// Totem rocks turn into normal rocks if not shot within 200 frames.
	if r.health == 5 && r.timer > 200 {
		r.Damage(g, 1, false)
	}

	colour := strconv.Itoa(max(g.wave, 0) % 3)
	health := strconv.Itoa(max(r.showHealth-1, 0))
	r.Image = "rock" + colour + strconv.Itoa(r.typ) + health
}
