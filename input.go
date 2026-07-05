package main

import "github.com/Zyko0/go-sdl3/sdl"

// keyDown reports whether a key is held, via the harness keyboard snapshot.
func keyDown(sc sdl.Scancode) bool { return app.Keyboard.Held(sc) }

func keyDown_left() bool  { return keyDown(sdl.SCANCODE_LEFT) }
func keyDown_right() bool { return keyDown(sdl.SCANCODE_RIGHT) }
func keyDown_up() bool    { return keyDown(sdl.SCANCODE_UP) }
func keyDown_down() bool  { return keyDown(sdl.SCANCODE_DOWN) }
func keyDown_space() bool { return keyDown(sdl.SCANCODE_SPACE) }
func keyDown_enter() bool { return keyDown(sdl.SCANCODE_RETURN) || keyDown(sdl.SCANCODE_KP_ENTER) }
