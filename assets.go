package whenbus

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:public all:static all:templates
var embeddedAssets embed.FS

func PublicFS() http.FileSystem {
	return mustSubFS("public")
}

func StaticFS() http.FileSystem {
	return mustSubFS("static")
}

func TemplatesFS() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}

func mustSubFS(dir string) http.FileSystem {
	sub, err := fs.Sub(embeddedAssets, dir)
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
