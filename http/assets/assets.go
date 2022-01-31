package assets

import "embed"

// Assets contains the web front-end static assets.
//
// Some valid accessors:
//  chesstempo/dist [dir]
//  chesstempo/dist/assets [dir]
//  chesstempo/dist/index.html [file]
//  chesstempo/dist/favicon.ico [file]
//
//go:embed chesstempo/dist/*
var FS embed.FS
