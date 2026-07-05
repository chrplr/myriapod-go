package main

import "math/rand"

type Bullet struct {
	Sprite
	done bool
}

func NewBullet(x, y float64) *Bullet {
	return &Bullet{Sprite: newSprite("bullet", x, y)}
}

func (b *Bullet) Update(g *Game) {
	// Move up the screen, 24 pixels per frame.
	b.Y -= 24

	// Damage a rock at the bullet's grid cell, if there is one.
	cx, cy := pos2cell(b.X, b.Y)
	if g.Damage(cx, cy, 1, true) {
		b.done = true
		return
	}

	// Otherwise check each segment and the flying enemy for a collision.
	for _, seg := range g.segments {
		if seg.CollidePoint(g.assets, b.X, b.Y) {
			g.explosions = append(g.explosions, NewExplosion(seg.X, seg.Y, 2))
			seg.health--

			// Possibly leave a rock where the segment died: health must be zero,
			// the cell must be empty, and the player must not overlap it.
			if seg.health == 0 && g.grid[seg.cellY][seg.cellX] == nil &&
				g.AllowMovement(g.player.X, g.player.Y, seg.cellX, seg.cellY) {
				g.grid[seg.cellY][seg.cellX] = NewRock(g, seg.cellX, seg.cellY, rand.Float64() < 0.2)
			}
			g.PlaySound("segment_explode", 1)
			g.score += 10
			b.done = true
			return
		}
	}

	if g.flyingEnemy != nil && g.flyingEnemy.CollidePoint(g.assets, b.X, b.Y) {
		g.explosions = append(g.explosions, NewExplosion(g.flyingEnemy.X, g.flyingEnemy.Y, 2))
		g.flyingEnemy.health--
		g.PlaySound("meanie_explode", 1)
		g.score += 20
		b.done = true
	}
}
