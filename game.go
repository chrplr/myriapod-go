package main

import (
	"math/rand"
	"sort"
	"strconv"
)

// Grid geometry: a cell is 32 pixels.
const (
	numGridRows = 25
	numGridCols = 14
)

// pos2cell converts pixel coordinates to grid-cell coordinates.
func pos2cell(x, y float64) (int, int) {
	return floorDiv(int(x)-16, 32), floorDiv(int(y), 32)
}

// cell2pos converts a grid cell to the pixel coordinates of its centre, plus an
// optional offset. The X axis has a 16px border on each side.
func cell2pos(cellX, cellY, xOffset, yOffset int) (float64, float64) {
	return float64(cellX*32 + 32 + xOffset), float64(cellY*32 + 16 + yOffset)
}

// cellKey builds a key for the occupied set. edge is -1 for a plain cell entry.
func cellKey(x, y, edge int) [3]int { return [3]int{x, y, edge} }

type Game struct {
	wave int
	time int

	player *Player

	grid        [][]*Rock
	bullets     []*Bullet
	explosions  []*Explosion
	segments    []*Segment
	flyingEnemy *FlyingEnemy

	occupied map[[3]int]bool
	score    int

	assets *Assets
	audio  *Audio
}

func NewGame(player *Player, assets *Assets, audio *Audio) *Game {
	grid := make([][]*Rock, numGridRows)
	for y := range grid {
		grid[y] = make([]*Rock, numGridCols)
	}
	return &Game{
		wave:     -1,
		player:   player,
		grid:     grid,
		occupied: make(map[[3]int]bool),
		assets:   assets,
		audio:    audio,
	}
}

// Damage applies damage to a rock at the given cell (if any), clearing the cell
// if the rock is destroyed. It returns whether a rock was present.
func (g *Game) Damage(cellX, cellY, amount int, fromBullet bool) bool {
	if cellX < 0 || cellX >= numGridCols || cellY < 0 || cellY >= numGridRows {
		return false
	}
	rock := g.grid[cellY][cellX]
	if rock != nil {
		if rock.Damage(g, amount, fromBullet) {
			g.grid[cellY][cellX] = nil
		}
	}
	return rock != nil
}

// AllowMovement reports whether the player sprite may occupy pixel (x, y). ax/ay,
// when non-negative, are a cell that must additionally be treated as blocked.
func (g *Game) AllowMovement(x, y float64, ax, ay int) bool {
	// Stay within the player's zone.
	if x < 40 || x > 440 || y < 592 || y > 784 {
		return false
	}
	x0, y0 := pos2cell(x-18, y-10)
	x1, y1 := pos2cell(x+18, y+10)
	for yi := y0; yi <= y1; yi++ {
		for xi := x0; xi <= x1; xi++ {
			if g.grid[yi][xi] != nil || (xi == ax && yi == ay) {
				return false
			}
		}
	}
	return true
}

// ClearRocksForRespawn destroys any rocks overlapping the player's respawn point.
func (g *Game) ClearRocksForRespawn(x, y float64) {
	x0, y0 := pos2cell(x-18, y-10)
	x1, y1 := pos2cell(x+18, y+10)
	for yi := y0; yi <= y1; yi++ {
		for xi := x0; xi <= x1; xi++ {
			g.Damage(xi, yi, 5, false)
		}
	}
}

func (g *Game) Update() {
	// Time drives segment movement; it advances twice as fast every fourth wave.
	if g.wave%4 == 3 {
		g.time += 2
	} else {
		g.time++
	}

	g.occupied = make(map[[3]int]bool)

	// Update order matches the original: bullets, segments, explosions, player,
	// flying enemy, then rocks.
	for _, b := range g.bullets {
		b.Update(g)
	}
	for _, s := range g.segments {
		s.Update(g)
	}
	for _, e := range g.explosions {
		e.Update(g)
	}
	if g.player != nil {
		g.player.Update(g)
	}
	if g.flyingEnemy != nil {
		g.flyingEnemy.Update(g)
	}
	for _, row := range g.grid {
		for _, rock := range row {
			if rock != nil {
				rock.Update(g)
			}
		}
	}

	// Drop finished bullets, explosions and dead segments.
	g.bullets = filter(g.bullets, func(b *Bullet) bool { return b.Y > 0 && !b.done })
	g.explosions = filter(g.explosions, func(e *Explosion) bool { return e.timer != 31 })
	g.segments = filter(g.segments, func(s *Segment) bool { return s.health > 0 })

	if g.flyingEnemy != nil {
		if g.flyingEnemy.health <= 0 || g.flyingEnemy.X < -35 || g.flyingEnemy.X > 515 {
			g.flyingEnemy = nil
		}
	} else if rand.Float64() < 0.01 {
		px := 240.0
		if g.player != nil {
			px = g.player.X
		}
		g.flyingEnemy = NewFlyingEnemy(px)
	}

	if len(g.segments) == 0 {
		g.startNextWaveOrAddRock()
	}
}

func (g *Game) startNextWaveOrAddRock() {
	numRocks := 0
	for _, row := range g.grid {
		for _, rock := range row {
			if rock != nil {
				numRocks++
			}
		}
	}

	if numRocks < 31+g.wave {
		// Not enough rocks yet - add one per frame at a random empty cell.
		for {
			x, y := rand.Intn(numGridCols), rand.Intn(numGridRows-3)+1
			if g.grid[y][x] == nil {
				g.grid[y][x] = NewRock(g, x, y, false)
				break
			}
		}
		return
	}

	// Enough rocks - spawn a new myriapod.
	g.PlaySound("wave", 1)
	g.wave++
	g.time = 0
	numSegments := 8 + g.wave/4*2 // 8 on the first four waves, then 10, ...
	healthTable := [4][2]int{{1, 1}, {1, 2}, {2, 2}, {1, 1}}
	for i := 0; i < numSegments; i++ {
		cellX, cellY := -1-i, 0
		health := healthTable[g.wave%4][i%2]
		fast := g.wave%4 == 3
		head := i == 0
		g.segments = append(g.segments, NewSegment(cellX, cellY, health, fast, head))
	}
}

func (g *Game) Draw() {
	g.assets.Blit("bg"+strconv.Itoa(max(g.wave, 0)%3), 0, 0)

	// Collect drawables: rocks, bullets, segments, explosions and the player.
	var objs []Drawable
	for _, row := range g.grid {
		for _, rock := range row {
			if rock != nil {
				objs = append(objs, rock)
			}
		}
	}
	for _, b := range g.bullets {
		objs = append(objs, b)
	}
	for _, s := range g.segments {
		objs = append(objs, s)
	}
	for _, e := range g.explosions {
		objs = append(objs, e)
	}
	if g.player != nil {
		objs = append(objs, g.player)
	}

	// Draw back-to-front by Y, with explosions always in front.
	sort.SliceStable(objs, func(i, j int) bool {
		ei, ej := isExplosion(objs[i]), isExplosion(objs[j])
		if ei != ej {
			return !ei // non-explosions first
		}
		return objs[i].PosY() < objs[j].PosY()
	})
	for _, obj := range objs {
		obj.Draw(g.assets)
	}

	// The flying enemy is always drawn on top.
	if g.flyingEnemy != nil {
		g.flyingEnemy.Draw(g.assets)
	}
}

// PlaySound plays a game sound, but only during play (never on the menu).
func (g *Game) PlaySound(name string, count int) {
	if g.player == nil {
		return
	}
	g.audio.PlaySound(name, count)
}

// Drawable is any on-screen object that can be depth-sorted and drawn.
type Drawable interface {
	Draw(a *Assets)
	PosY() float64
}

func isExplosion(d Drawable) bool {
	_, ok := d.(*Explosion)
	return ok
}

func filter[T any](s []T, keep func(T) bool) []T {
	out := s[:0]
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
