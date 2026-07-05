package main

// Glue between the game and the pgzgo harness: the embedded assets (the
// //go:embed directives must live in this package) and the running App that
// input.go reads its keyboard snapshot from.

import (
	"embed"

	"github.com/chrplr/pgzgo"
)

// Assets and Audio are the game's names for the harness drawing surface and mixer.
type Assets = pgzgo.Screen
type Audio = pgzgo.Audio

//go:embed images
var imagesFS embed.FS

//go:embed sounds music
var audioFS embed.FS

// app is the running harness; input.go reads its keyboard snapshot.
var app *pgzgo.App
