package main

// Sprite is a positioned image with a Pygame-Zero-style anchor. By default the
// anchor is the image centre; a sprite may instead pin an explicit pixel offset
// within the image to its position (used by tall "totem" rocks).
type Sprite struct {
	X, Y  float64
	Image string

	// If anchorCentre is true the image centre sits at (X, Y); otherwise the
	// pixel (ax, ay) within the image sits at (X, Y).
	anchorCentre bool
	ax, ay       float64
}

func newSprite(image string, x, y float64) Sprite {
	return Sprite{X: x, Y: y, Image: image, anchorCentre: true}
}

// anchorOffset returns the pixel offset of the anchor from the image top-left.
func (s *Sprite) anchorOffset(a *Assets) (float64, float64) {
	if s.anchorCentre {
		w, h := a.Size(s.Image)
		return w / 2, h / 2
	}
	return s.ax, s.ay
}

func (s *Sprite) Draw(a *Assets) {
	ax, ay := s.anchorOffset(a)
	a.Blit(s.Image, s.X-ax, s.Y-ay)
}

// PosY is the sprite's Y position, used to order drawing back-to-front.
func (s *Sprite) PosY() float64 { return s.Y }

// CollidePoint reports whether (px, py) lies within the sprite's current image
// rectangle, matching Pygame's Rect.collidepoint.
func (s *Sprite) CollidePoint(a *Assets, px, py float64) bool {
	ax, ay := s.anchorOffset(a)
	w, h := a.Size(s.Image)
	left, top := s.X-ax, s.Y-ay
	return px >= left && px < left+w && py >= top && py < top+h
}
