package main

import "strconv"

const (
	invulnerabilityTime = 100
	respawnTime         = 100
	reloadTime          = 10
)

type Player struct {
	Sprite
	direction int
	frame     int
	lives     int
	alive     bool
	timer     int
	fireTimer int
}

func NewPlayer(x, y float64) *Player {
	return &Player{
		Sprite: newSprite("blank", x, y),
		lives:  3,
		alive:  true,
	}
}

// move steps up to speed pixels in (dx, dy), stopping if a cell is blocked.
func (p *Player) move(g *Game, dx, dy, speed int) {
	for i := 0; i < speed; i++ {
		if g.AllowMovement(p.X+float64(dx), p.Y+float64(dy), -1, -1) {
			p.X += float64(dx)
			p.Y += float64(dy)
		}
	}
}

func (p *Player) Update(g *Game) {
	p.timer++

	if p.alive {
		dx := 0
		if keyDown_left() {
			dx = -1
		} else if keyDown_right() {
			dx = 1
		}
		dy := 0
		if keyDown_up() {
			dy = -1
		} else if keyDown_down() {
			dy = 1
		}

		// The 3-abs(...) keeps diagonal speed roughly equal to axis-aligned speed.
		p.move(g, dx, 0, 3-abs(dy))
		p.move(g, 0, dy, 3-abs(dx))

		// Rotate the sprite towards the direction of travel, one step every other
		// frame, so turns look gradual. See the original for the table rationale.
		directions := []int{7, 0, 1, 6, -1, 2, 5, 4, 3}
		dir := directions[dx+3*dy+4]
		if p.timer%2 == 0 && dir >= 0 {
			difference := dir - p.direction
			rotationTable := []int{0, 1, 1, -1}
			rotation := rotationTable[pmod(difference, 4)]
			p.direction = pmod(p.direction+rotation, 4)
		}

		p.fireTimer--

		// Fire the cannon (or let the firing animation finish).
		if p.fireTimer < 0 && (p.frame > 0 || keyDown_space()) {
			if p.frame == 0 {
				g.PlaySound("laser", 1)
				g.bullets = append(g.bullets, NewBullet(p.X, p.Y-8))
			}
			p.frame = (p.frame + 1) % 3
			p.fireTimer = reloadTime
		}

		// Collide with enemy segments and the flying enemy.
		if g.flyingEnemy != nil {
			p.checkCollision(g, &g.flyingEnemy.Sprite)
		}
		for _, seg := range g.segments {
			if !p.alive {
				break
			}
			p.checkCollision(g, &seg.Sprite)
		}
	} else if p.timer > respawnTime {
		// Respawn.
		p.alive = true
		p.timer = 0
		p.X, p.Y = 240, 768
		g.ClearRocksForRespawn(p.X, p.Y)
	}

	// Draw the sprite when alive; flicker while briefly invulnerable after respawn.
	invulnerable := p.timer > invulnerabilityTime
	if p.alive && (invulnerable || p.timer%2 == 0) {
		p.Image = "player" + strconv.Itoa(p.direction) + strconv.Itoa(p.frame)
	} else {
		p.Image = "blank"
	}
}

func (p *Player) checkCollision(g *Game, enemy *Sprite) {
	if !p.alive {
		return
	}
	if enemy.CollidePoint(g.assets, p.X, p.Y) && p.timer > invulnerabilityTime {
		g.PlaySound("player_explode", 1)
		g.explosions = append(g.explosions, NewExplosion(p.X, p.Y, 1))
		p.alive = false
		p.timer = 0
		p.lives--
	}
}
