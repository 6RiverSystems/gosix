package swaggerui

import (
	"embed"
	"io/fs"
)

// two stage, because the go:embed generated FS will have a directory prefix we
// need to strip
//go:embed ui/*
var uiFS embed.FS

var FS fs.FS

func init() {
	var err error
	FS, err = fs.Sub(uiFS, "ui")
	if err != nil {
		panic(err)
	}
}
