package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"kbb1.com/fileindex"
	"kbb1.com/fileutils"

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
		Path string `json:"path" form:"path"`
	}
)

var (
	fileMap syncmap.Map
	srvConf ServerConf
)

func getfs() *fileindex.FastSearch {
	return srvConf.Index.GetFS()
}

func search(sha1 string) (fileindex.FileList, bool) {
	return srvConf.Index.GetFS().Search(sha1)
}

func setfs(fs *fileindex.FastSearch) {
	srvConf.Index.SetFS(fs)
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
		if fl, ok := search(sha1sum); ok {
			c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
			c.Response().Header().Set(echo.HeaderContentDisposition, "attachment")
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
		if _, ok := search(r.SHA1); ok {
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

// POST /api/v1/update
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

// GET /api/v1/catalog
func getCatalog(c echo.Context) (err error) {
	buf := new(bytes.Buffer)
	files := getfs().GetAll()
	for _, fl := range files {
		storages := make([]string, 0, len(fl))
		for _, fr := range fl {
			if fr.Device == nil {
				fmt.Println("Wrong device:", fr.Path)
			} else {
				storages = append(storages, fr.Device.Id)
			}
		}
		storagesJson, _ := json.Marshal(storages)
		fmt.Fprintf(buf, "%s,%s\n", fl[0].Sha1, storagesJson)
	}

	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
	return c.Blob(http.StatusOK, "text/plain", buf.Bytes())
}

// GET /api/v1/storages
func getStorages(c echo.Context) (err error) {
	ll := make([]fileindex.Storage, 0, 100)
	storages.Range(func(key, value interface{}) bool {
		st := value.(*fileindex.Storage)
		ll = append(ll, *st)
		return true
	})
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
	return c.JSON(http.StatusOK, ll)
}

func webServer(conf ServerConf) {
	srvConf = conf

	e := echo.New()
	e.HideBanner = true

	e.GET("/", getHello)
	e.GET("/get/:sha1/:name", getFile)

	e.GET("/api/v1/catalog", getCatalog)
	e.POST("/api/v1/get", postRegFile)
	e.GET("/api/v1/storages", getStorages)
	e.POST("/api/v1/update", postUpdate)

	e.Logger.Fatal(e.Start(srvConf.Listen))
}

func pathTranslate(path string) string {
	path = strings.Replace(path, "\\", "/", -1)
	if x := strings.Index(path, "/Archive/"); x >= 0 {
		path = srvConf.BasePathArchive + path[x:]
	} else if x := strings.Index(path, "/Archive_PN/"); x >= 0 {
		path = srvConf.BasePathArchive + path[x:]
	} else if x := strings.Index(path, "/__BACKUP/"); x >= 0 {
		path = "/net/server/original" + path[x:]
	} else {
		path = ""
	}
	return path
}

func updateServer(conf UpdateConf) {
	tick := time.Tick(conf.Reload)
	for {
		select {
		case <-tick:
			if conf.Index.IsModified() {
				log.Println("Debug: Reload indexes")
				conf.Index.Load()
			}
		case path := <-conf.Update:
			pathtr := pathTranslate(path)
			if pathtr == "" {
				log.Println("Update (unknown path):", path)
				continue
			}
			log.Println("Update:", pathtr)

			if stat, err := os.Lstat(pathtr); err == nil {
				mtime := stat.ModTime().Unix()
				size := stat.Size()

				fs := getfs()
				fr, ok := fs.SearchPath(pathtr)
				if ok && fr.Size == size && fr.Mtime == mtime {
					continue
				}

				fr = &fileindex.FileRec{
					Path:  pathtr,
					Size:  size,
					Mtime: mtime,
				}
				if !filter(fr, nil) {
					continue
				}

				sha1, _, stat2, err := fileutils.SHA1_File(pathtr)
				if err != nil {
					log.Println(err)
					continue
				}

				if stat2.Size() != stat.Size() && stat2.ModTime() != stat2.ModTime() {
					log.Println("Update (being modified):", pathtr)
					continue
				}

				fr.Sha1 = hex.EncodeToString(sha1)
				log.Println("SHA1:", fr.Sha1)

				fsdup := fs.Duplicate()
				fsdup.Update(fr)
				setfs(fsdup)
			} else {
				log.Println(err)
			}
		}
	}
}
