package main

import (
	"net/http"

	"github.com/labstack/echo"
)

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!\n")
}

func getfile(c echo.Context) error {
	sha1sum := c.Param("sha1")

	index.Lock()
	sha1 := index.SHA1
	index.Unlock()

	if fl, ok := sha1.Search(sha1sum); ok {
		return c.File(fl[0].Path)
	} else {
		return c.NoContent(http.StatusNotFound)
	}
}

func server(listen string) {
	e := echo.New()
	e.HideBanner = true

	e.GET("/", hello)
	e.GET("/get/:sha1/:name", getfile)

	e.Logger.Fatal(e.Start(listen))
}
