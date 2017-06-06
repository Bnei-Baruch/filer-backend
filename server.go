package main

import (
	"log"
	"net/http"
	"time"

	"kbb1.com/fileindex"

	"github.com/labstack/echo"
	"golang.org/x/sync/syncmap"
)

type (
	RegFileReq struct {
		SHA1     string `json:"sha1" form:"sha1"`
		Name     string `json:"name" form:"name"`
		ClientIP string `json:"clientip" form:"clientip"`
	}

	RegFileResp struct {
		URL string `json:"url" form:"url"`
	}

	UpdateReq struct {
		Path string `json:"path" form:"url"`
	}
)

var (
	fileMap syncmap.Map
	srvConf ServerConf
)

func getfs() (fs fileindex.FastSearch) {
	srvConf.Index.Lock()
	fs = srvConf.Index.FS
	srvConf.Index.Unlock()
	return
}

func getHello(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
	return c.String(http.StatusOK, "Hello, World!\n")
}

func getFile(c echo.Context) error {
	sha1sum := c.Param("sha1")
	name := c.Param("name")
	key := sha1sum + "/" + name
	if _, ok := fileMap.Load(key); ok {
		fs := getfs()
		if fl, ok := fs.Search(sha1sum); ok {
			log.Println(fl[0].Path)
			c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
			return c.File(fl[0].Path)
		}
	}
	return c.NoContent(http.StatusNotFound)
}

func postRegFile(c echo.Context) (err error) {
	r := new(RegFileReq)
	if err = c.Bind(r); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	if r.SHA1 != "" && r.Name != "" {
		fs := getfs()
		if _, ok := fs.Search(r.SHA1); ok {
			key := r.SHA1 + "/" + r.Name
			fileMap.Store(key, time.Now().Unix())
			res := new(RegFileResp)
			res.URL = srvConf.BaseURL + key
			c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
			return c.JSON(http.StatusOK, res)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func postUpdate(c echo.Context) (err error) {
	r := new(UpdateReq)
	if err = c.Bind(r); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	if r.Path == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	if srvConf.Update != nil {
		srvConf.Update <- r.Path
	}
	return c.NoContent(http.StatusOK)
}

func webServer(conf ServerConf) {
	srvConf = conf

	e := echo.New()
	e.HideBanner = true

	e.GET("/", getHello)
	e.GET("/get/:sha1/:name", getFile)

	e.POST("/api/v1/get", postRegFile)
	e.POST("/api/v1/update", postUpdate)

	e.Logger.Fatal(e.Start(srvConf.Listen))
}

func updateServer(conf UpdateConf) {
	tick := time.Tick(conf.Reload)
	for {
		select {
		case <-tick:
			if conf.Index.IsModified() {
				log.Println("Reload indexes")
				conf.Index.Load()
			}
		case path := <-conf.Update:
			log.Println("Update:", path)
		}
	}
}
