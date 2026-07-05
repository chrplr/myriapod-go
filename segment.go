package main

import "strconv"

// Direction constants and their grid deltas.
const (
	dirUp    = 0
	dirRight = 1
	dirDown  = 2
	dirLeft  = 3
)

var (
	dxTable = [4]int{0, 1, 0, -1}
	dyTable = [4]int{-1, 0, 1, 0}

	// secondaryAxisSpeed: how far the segment moves along the secondary axis at
	// each of the 16 phases when it makes two 45° turns within a cell.
	secondaryAxisSpeed = buildSecondaryAxisSpeed()
	// secondaryAxisPositions[i] is the cumulative secondary-axis movement before phase i.
	secondaryAxisPositions = buildSecondaryAxisPositions()

	// Rotation matrices selected by in_edge, applied to the base (top-to-bottom) offset.
	rotationMatrices = [4][4]int{
		{1, 0, 0, 1},
		{0, -1, 1, 0},
		{-1, 0, 0, -1},
		{0, 1, -1, 0},
	}
)

func buildSecondaryAxisSpeed() [16]int {
	var s [16]int
	for i := 4; i < 12; i++ {
		s[i] = 1
	}
	for i := 12; i < 16; i++ {
		s[i] = 2
	}
	return s
}

func buildSecondaryAxisPositions() [16]int {
	var p [16]int
	sum := 0
	for i := 0; i < 16; i++ {
		p[i] = sum
		sum += secondaryAxisSpeed[i]
	}
	return p
}

func inverseDirection(dir int) int {
	switch dir {
	case dirUp:
		return dirDown
	case dirRight:
		return dirLeft
	case dirDown:
		return dirUp
	default:
		return dirRight
	}
}

func isHorizontal(dir int) bool {
	return dir == dirLeft || dir == dirRight
}

// Segment is one link of a myriapod. It moves cell-to-cell in a fixed 16-phase
// cycle, choosing which edge to leave through at phase 4.
type Segment struct {
	Sprite
	cellX, cellY       int
	health             int
	fast               bool
	head               bool
	inEdge, outEdge    int
	disallowDirection  int
	previousXDirection int
}

func NewSegment(cx, cy, health int, fast, head bool) *Segment {
	return &Segment{
		Sprite:             newSprite("blank", 0, 0),
		cellX:              cx,
		cellY:              cy,
		health:             health,
		fast:               fast,
		head:               head,
		inEdge:             dirLeft,
		outEdge:            dirRight,
		disallowDirection:  dirUp,
		previousXDirection: 1,
	}
}

// rankKey packs the ordering factors for a proposed out edge into a single
// integer (most significant factor in the highest bit). Lower is preferred,
// matching the tuple comparison used with Python's min().
func (s *Segment) rankKey(g *Game, edge int) int {
	newCellX := s.cellX + dxTable[edge]
	newCellY := s.cellY + dyTable[edge]

	out := newCellX < 0 || newCellX > numGridCols-1 || newCellY < 0 || newCellY > numGridRows-1
	turningBack := edge == s.inEdge
	disallowed := edge == s.disallowDirection

	var rockPresent bool
	if !(out || (newCellY == 0 && newCellX < 0)) {
		rockPresent = g.grid[newCellY][newCellX] != nil
	}

	occupiedBySegment := g.occupied[cellKey(newCellX, newCellY, -1)] ||
		g.occupied[cellKey(s.cellX, s.cellY, edge)]

	// Prefer horizontal movement unless a rock blocks it; if blocked both ways,
	// prefer vertical.
	var horizontalBlocked bool
	if rockPresent {
		horizontalBlocked = isHorizontal(edge)
	} else {
		horizontalBlocked = !isHorizontal(edge)
	}

	sameAsPreviousX := edge == s.previousXDirection

	return b(out)<<6 | b(turningBack)<<5 | b(disallowed)<<4 | b(occupiedBySegment)<<3 |
		b(rockPresent)<<2 | b(horizontalBlocked)<<1 | b(sameAsPreviousX)
}

// bestOutEdge returns the lowest-ranked (most preferred) edge, ties broken by
// the lowest edge number, matching Python's min(range(4), key=...).
func (s *Segment) bestOutEdge(g *Game) int {
	best, bestKey := 0, s.rankKey(g, 0)
	for edge := 1; edge < 4; edge++ {
		if k := s.rankKey(g, edge); k < bestKey {
			best, bestKey = edge, k
		}
	}
	return best
}

func (s *Segment) Update(g *Game) {
	phase := g.time % 16

	if phase == 0 {
		// Entering a new cell.
		s.cellX += dxTable[s.outEdge]
		s.cellY += dyTable[s.outEdge]
		s.inEdge = inverseDirection(s.outEdge)

		// Bounce between the bottom of the screen and row 18 to stay a threat.
		// On the title screen (no player) segments may return to the top.
		topRow := 0
		if g.player != nil {
			topRow = 18
		}
		if s.cellY == topRow {
			s.disallowDirection = dirUp
		}
		if s.cellY == numGridRows-1 {
			s.disallowDirection = dirDown
		}
	} else if phase == 4 {
		// Decide which edge (and cell) to leave through.
		s.outEdge = s.bestOutEdge(g)
		if isHorizontal(s.outEdge) {
			s.previousXDirection = s.outEdge
		}

		newCellX := s.cellX + dxTable[s.outEdge]
		newCellY := s.cellY + dyTable[s.outEdge]

		// Destroy any rock in the target cell.
		if newCellX >= 0 && newCellX < numGridCols {
			g.Damage(newCellX, newCellY, 5, false)
		}

		// Reserve the target cell so other segments avoid it.
		g.occupied[cellKey(newCellX, newCellY, -1)] = true
		g.occupied[cellKey(newCellX, newCellY, inverseDirection(s.outEdge))] = true
	}

	// Movement within the cell. See the original for the derivation.
	turnIdx := pmod(s.outEdge-s.inEdge, 4)
	offsetX := secondaryAxisPositions[phase] * (2 - turnIdx)
	stolenY := (turnIdx % 2) * secondaryAxisPositions[phase]
	offsetY := -16 + (phase * 2) - stolenY

	rm := rotationMatrices[s.inEdge]
	offsetX, offsetY = offsetX*rm[0]+offsetY*rm[1], offsetX*rm[2]+offsetY*rm[3]

	s.X, s.Y = cell2pos(s.cellX, s.cellY, offsetX, offsetY)

	// Choose the sprite: seg<fast><health==2><head><direction 0-7><legFrame 0-3>.
	direction := pmod(secondaryAxisSpeed[phase]*(turnIdx-2)+(s.inEdge*2)+4, 8)
	legFrame := phase / 4
	s.Image = "seg" + b2s(s.fast) + b2s(s.health == 2) + b2s(s.head) +
		strconv.Itoa(direction) + strconv.Itoa(legFrame)
}
