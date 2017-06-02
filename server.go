package main

import (
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"golang.org/x/sync/syncmap"
)

type (
	RegFile struct {
		SHA1     string `json:"sha1" form:"sha1"`
		Name     string `json:"name" form:"name"`
		ClientIP string `json:"clientip" form:"clientip"`
	}
	RegFileResp struct {
		URL string `json:"url" form:"url"`
	}
)

var (
	FileMap syncmap.Map
)

func hello(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
	return c.String(http.StatusOK, "Hello, World!\n")
}

func getfile(c echo.Context) error {
	sha1sum := c.Param("sha1")
	name := c.Param("name")
	key := sha1sum + "/" + name
	if _, ok := FileMap.Load(key); ok {
		index.Lock()
		sha1 := index.SHA1
		index.Unlock()

		if fl, ok := sha1.Search(sha1sum); ok {
			log.Println(fl[0].Path)
			c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
			return c.File(fl[0].Path)
		}
	}
	return c.NoContent(http.StatusNotFound)
}

func regfile(c echo.Context) (err error) {
	r := new(RegFile)
	if err = c.Bind(r); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	if r.SHA1 != "" && r.Name != "" {
		index.Lock()
		sha1 := index.SHA1
		index.Unlock()
		if _, ok := sha1.Search(r.SHA1); ok {
			key := r.SHA1 + "/" + r.Name
			FileMap.Store(key, time.Now().Unix())
			res := new(RegFileResp)
			res.URL = "http://files.kabbalahmedia.info/get/" + key
			c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
			return c.JSON(http.StatusOK, res)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func server(listen string) {
	e := echo.New()
	e.HideBanner = true

	e.GET("/", hello)
	e.GET("/get/:sha1/:name", getfile)

	e.POST("/api/v1/get", regfile)

	e.Logger.Fatal(e.Start(listen))
}
