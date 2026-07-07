# Myriapod — Python vs. Go implementation comparison

This document compares the Go port in this folder with the original
`myriapod.py`. Myriapod is a *Centipede*-style shooter on a 14×25 grid: a
segmented myriapod winds through the rocks, the player moves in a bottom band and
shoots upward, and rocks/totems/flying enemies fill out the board. It has by far
the most intricate logic of the ports — the segment pathfinding in particular —
so it best shows how idiomatic Python constructs map onto Go. The Go version is a
faithful translation; the differences are mechanical consequences of Go's static
typing, the lack of class inheritance, and swapping Pygame Zero for
[go-sdl3](https://github.com/Zyko0/go-sdl3).

---

## 1. File organisation

Python is a single 911-line module. The Go port splits it by entity:

| Python section | Go file |
|---|---|
| `update()`, `draw()`, state machine, helpers, mixer setup | `main.go` |
| `Game` class (grid, waves, update/draw, damage) | `game.go` |
| `Segment` (+ movement constants, `rank`) | `segment.go` |
| `Player` | `player.go` |
| `Bullet` | `bullet.go` |
| `Rock` | `rock.go` |
| `FlyingEnemy` | `flyingenemy.go` |
| `Explosion` | `explosion.go` |
| `Actor` behaviour / anchors / collision | `sprite.go` |
| image cache | pgzgo `Screen` (harness) |
| `play_sound` / music | pgzgo `Audio` (harness) |
| keyboard reading | `input.go` |

---

## 2. Inheritance → struct embedding

Python subclasses Pygame Zero's `Actor` for every entity
(`Explosion`, `Player`, `FlyingEnemy`, `Rock`, `Bullet`, `Segment`). Go embeds a
`Sprite` struct that also reproduces Pygame's anchor model, because "totem" rocks
use a non-centre anchor:

```go
type Sprite struct {
    X, Y         float64
    Image        string
    anchorCentre bool
    ax, ay       float64
}
func (s *Sprite) CollidePoint(a *Assets, px, py float64) bool { ... } // Rect.collidepoint
func (s *Sprite) PosY() float64 { return s.Y }                        // for depth sort
```

Each `update(self)` becomes a method taking the game as a parameter
(`func (s *Segment) Update(g *Game)`), replacing the Python global `game`.

---

## 3. The star translation: `occupied` tuple-set → `map[[3]int]bool`

Python's segment reservation uses a `set` that mixes **two shapes of tuple**:

```python
self.occupied = set()
game.occupied.add((new_cell_x, new_cell_y))                            # 2-tuple
game.occupied.add((new_cell_x, new_cell_y, inverse_direction(...)))    # 3-tuple
...
occupied_by_segment = (new_cell_x, new_cell_y) in game.occupied \
                   or (self.cell_x, self.cell_y, proposed_out_edge) in game.occupied
```

Go maps need a single, fixed key type. The port unifies both shapes into a
`[3]int` key, using a sentinel `-1` in the third slot for the "plain cell" case:

```go
func cellKey(x, y, edge int) [3]int { return [3]int{x, y, edge} }
occupied map[[3]int]bool
...
g.occupied[cellKey(newCellX, newCellY, -1)] = true
g.occupied[cellKey(newCellX, newCellY, inverseDirection(s.outEdge))] = true
...
occupiedBySegment := g.occupied[cellKey(newCellX, newCellY, -1)] ||
                     g.occupied[cellKey(s.cellX, s.cellY, edge)]
```

A comparable `[3]int` array is a valid Go map key, so this reproduces the set
semantics exactly while staying statically typed.

---

## 4. The other star: `rank()` closure + `min(key=…)` → bit-packed integer

This is the cleverest part of the original and of the port. Python's `rank`
returns a closure that returns a **tuple of seven booleans**, which
`min(range(4), key=self.rank())` compares lexicographically (with `False < True`):

```python
def inner(proposed_out_edge):
    ...
    return (out, turning_back_on_self, direction_disallowed,
            occupied_by_segment, rock_present, horizontal_blocked,
            same_as_previous_x_direction)
self.out_edge = min(range(4), key=self.rank())
```

Go can't order tuples out of the box. The insight is that **lexicographic
comparison of a tuple of booleans is identical to integer comparison of those
bits packed most-significant-first.** So the port packs the seven factors into
one `int` and does a manual `min`:

```go
func (s *Segment) rankKey(g *Game, edge int) int {
    ...
    return b(out)<<6 | b(turningBack)<<5 | b(disallowed)<<4 | b(occupiedBySegment)<<3 |
           b(rockPresent)<<2 | b(horizontalBlocked)<<1 | b(sameAsPreviousX)
}
func (s *Segment) bestOutEdge(g *Game) int {
    best, bestKey := 0, s.rankKey(g, 0)
    for edge := 1; edge < 4; edge++ {
        if k := s.rankKey(g, edge); k < bestKey { best, bestKey = edge, k }
    }
    return best
}
```

The strict `<` keeps the earliest edge on ties, matching Python's `min` (which
returns the first minimal element). The result is byte-for-byte the same edge
choice each frame.

---

## 5. `sum(self.grid, …)` flatten + heterogeneous list → typed collections

Python flattens the 2-D grid and other lists into one mixed list of `Actor`s and
`None`s, both for updating and for depth-sorted drawing:

```python
all_objects = sum(self.grid, self.bullets + self.segments + self.explosions
                             + [self.player] + [self.flying_enemy])
for obj in all_objects:
    if obj: obj.update()
```

Go can't hold "any Actor or None" in a typed slice, so:

- **Update** iterates each typed slice in the same order the Python `sum`
  produced (bullets → segments → explosions → player → flying enemy → grid
  rocks). This ordering matters: segments mutate the grid and the `occupied` set
  as they update, so the port preserves it exactly.
- **Draw** builds a `[]Drawable` (an interface with `Draw` and `PosY`) of the
  non-nil objects and depth-sorts it. The `isinstance(obj, Explosion)` sort key
  becomes a type assertion:

  ```go
  type Drawable interface { Draw(a *Assets); PosY() float64 }
  func isExplosion(d Drawable) bool { _, ok := d.(*Explosion); return ok }

  sort.SliceStable(objs, func(i, j int) bool {
      ei, ej := isExplosion(objs[i]), isExplosion(objs[j])
      if ei != ej { return !ei }               // non-explosions first
      return objs[i].PosY() < objs[j].PosY()   // then by Y
  })
  ```

Likewise `isinstance(obj, Segment)` in the bullet code becomes iterating the
typed `g.segments` slice directly.

---

## 6. List comprehensions → a generic `filter`

Python prunes finished objects with comprehensions:

```python
self.bullets    = [b for b in self.bullets if b.y > 0 and not b.done]
self.explosions = [e for e in self.explosions if not e.timer == 31]
self.segments   = [s for s in self.segments if s.health > 0]
```

Go uses a single generic helper that filters in place:

```go
func filter[T any](s []T, keep func(T) bool) []T {
    out := s[:0]
    for _, v := range s { if keep(v) { out = append(out, v) } }
    return out
}
...
g.bullets  = filter(g.bullets,  func(b *Bullet) bool { return b.Y > 0 && !b.done })
g.segments = filter(g.segments, func(s *Segment) bool { return s.health > 0 })
```

Go generics let one helper cover all three element types.

---

## 7. Python integer semantics: `//` and `%`

The grid maths depends on Python's **floor** division and **floor** modulo, which
differ from Go's truncating operators for negative operands. Segments legitimately
sit at negative cell coordinates when they walk on from off-screen, and
`out_edge - in_edge` can be negative, so the port provides explicit helpers:

```go
func floorDiv(a, b int) int { q := a / b; if a%b != 0 && (a<0) != (b<0) { q-- }; return q }
func pmod(a, b int) int      { m := a % b; if m < 0 { m += b }; return m }
```

Used wherever Python relied on the floor behaviour:

| Python | Go |
|---|---|
| `(int(x)-16)//32` in `pos2cell` | `floorDiv(int(x)-16, 32)` |
| `turn_idx = (out_edge - in_edge) % 4` | `pmod(s.outEdge-s.inEdge, 4)` |
| `rotation_table[difference % 4]` | `rotationTable[pmod(difference, 4)]` |
| `direction = (... ) % 8` | `pmod(..., 8)` |

Getting these wrong would corrupt the segment's cell tracking — this is the most
important numeric subtlety in the whole port.

---

## 8. Small type conversions

- **bool → int/str.** Python does `str(int(self.fast))` to build sprite names.
  Go provides `b(bool) int` and `b2s(bool) string`:

  ```go
  s.Image = "seg" + b2s(s.fast) + b2s(s.health == 2) + b2s(s.head) +
            strconv.Itoa(direction) + strconv.Itoa(legFrame)
  ```

- **Generic `abs`.** `func abs[T int | float64](x T) T` serves both the integer
  movement maths and any float use.

- **Negative string indexing.** Python's `score[-i]` (digits right-to-left)
  becomes `score[len(score)-i]` in the score display.

- **Grid initialisation.** `[[None]*cols for _ in range(rows)]` becomes a
  `make`d `[][]*Rock` whose zero value (`nil`) already means "empty cell".

---

## 9. Movement maths kept verbatim

The segment's per-cell motion — `SECONDARY_AXIS_SPEED`/`SECONDARY_AXIS_POSITIONS`
tables, the `stolen_y_movement` primary/secondary-axis trick, the four
`rotation_matrix` variants, and the `direction`/`leg_frame` sprite selection —
are ported line for line, with the tables built at init:

```go
var rotationMatrices = [4][4]int{{1,0,0,1},{0,-1,1,0},{-1,0,0,-1},{0,1,-1,0}}
...
offsetX, offsetY = offsetX*rm[0]+offsetY*rm[1], offsetX*rm[2]+offsetY*rm[3]
```

Note Go's simultaneous assignment reproduces Python's tuple assignment for the
rotation, so the right-hand side uses the pre-update `offsetX`/`offsetY`.

---

## 10. State, input, sound, loop

- **State enum** → `const … iota`; **space-press edge detection** preserved with a
  package-level `spaceDown`.
- **Optional player** (`self.player = None` on the menu/attract demo) → `*Player`
  nil; `play_sound` and various branches gate on `g.player != nil`.
- **Input**: Python's `keyboard.left` etc. → pgzgo's `app.Keyboard.Held(sc)` snapshot, wrapped by `keyDown_left()`-style helpers.
- **Sound/music**: `getattr(sounds, name+idx).play()` → preloaded
  `map[string]*mixer.Audio` + `PlayAudio`; `music.play("theme")` → a looping track.
- **Loop**: `pgzrun.go()` → pgzgo's `app.Loop`, a fixed-step, FPS-capped loop. As in the
  original, game time (not real time) drives everything, and it advances at double
  rate every fourth wave (`g.time += 2`).

---

## 11. What is intentionally identical

- Grid geometry (`pos2cell`/`cell2pos`), the 32px cells and 16px borders.
- Segment pathfinding, per-cell 16-phase movement, turn handling, sprite
  selection, and the bottom-band / row-18 bouncing behaviour.
- Player movement (with the `3 - abs(...)` diagonal-speed compensation), gradual
  turning table, firing/reload, respawn, invulnerability flicker, lives.
- Rock health/growing animation, totems (5 health, bonus, 200-frame decay),
  bullet/segment rock destruction and 20%-totem spawning.
- Wave progression, rock-count gating, segment counts/health/fast/head per wave,
  flying-enemy spawn logic, scoring, and all UI (lives, score, menu, game over).

---

## 12. Summary of differences

| Category | Difference | Reason |
|---|---|---|
| Inheritance | `Actor` subclasses → embedded `Sprite` | no classes |
| Tuple set | mixed 2-/3-tuple `set` → `map[[3]int]bool` + `-1` sentinel | fixed map key type |
| `min(key=closure)` | tuple-of-bools compare → bit-packed `int` + manual min | no tuple ordering |
| Heterogeneous list | `sum(grid, …)` of Actors/None → typed slices + `Drawable` | static typing |
| Comprehensions | `[x for x…]` → generic `filter[T]` | idiom + generics |
| Integer maths | `//`, `%` → `floorDiv`, `pmod` | Go truncates toward zero |
| Conversions | `str(int(bool))`, `score[-i]` → `b2s`, `score[len-i]` | no bool→int/neg-index |
| Optional player | `None` → `*Player` nil | no `None` |
| Framework | Pygame Zero → pgzgo (over go-sdl3) | library swap |

The grid rules, segment AI, wave design, and all the movement/rotation maths are
line-by-line equivalent to `myriapod.py`. The two most interesting translations —
the `[3]int` occupied-set key and the bit-packed `rank` comparison — are cases
where a Go idiom expresses a dynamically typed Python construct exactly.
