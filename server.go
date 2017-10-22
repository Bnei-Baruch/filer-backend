package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"kbb1.com/fileindex"
	"kbb1.com/fileutils"

	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
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

	TranscodeReq struct {
		SHA1   string `json:"sha1" form:"sha1"`
		Format string `json:"format" form:"format"`
	}

	UpdateReq struct {
		Path string `json:"path" form:"path"`
	}
)

var (
	fileMap sync.Map
	srvCtx  ServerCtx
)

func getfs() *fileindex.FastSearch {
	return srvCtx.Index.GetFS()
}

func search(sha1 string) (fileindex.FileList, bool) {
	return srvCtx.Index.GetFS().Search(sha1)
}

func setfs(fs *fileindex.FastSearch) {
	srvCtx.Index.SetFS(fs)
}

func getHello(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")
	return c.String(http.StatusOK, "Hello, World!\n")
}

// GET /get/:sha1/:name
func getFile(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")

	sha1sum := c.Param("sha1")
	name := c.Param("name")
	key := sha1sum + "/" + name
	if _, ok := fileMap.Load(key); ok {
		if fl, ok := search(sha1sum); ok {
			c.Response().Header().Set(echo.HeaderContentDisposition, "attachment")
			return c.File(fl[0].Path)
		}
	}
	return c.NoContent(http.StatusNotFound)
}

// POST /api/v1/get
func postRegFile(c echo.Context) (err error) {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")

	r := new(RegFileReq)
	if err = c.Bind(r); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	if r.SHA1 != "" && r.Name != "" {
		if _, ok := search(r.SHA1); ok {
			key := r.SHA1 + "/" + r.Name
			fileMap.Store(key, time.Now().Unix())
			res := new(RegFileResp)
			res.URL = srvCtx.Config.BaseURL + key
			return c.JSON(http.StatusOK, res)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /api/v1/transcode
func postTranscode(c echo.Context) (err error) {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")

	r := new(TranscodeReq)
	if err = c.Bind(r); err != nil {
		return c.String(http.StatusBadRequest, "Wrong parameters")
	}
	if r.SHA1 == "" || r.Format != "mp4" {
		return c.String(http.StatusBadRequest, "Wrong parameters")
	}

	if fl, ok := search(r.SHA1); ok {
		var task TranscodeTask
		task.Source = fl[0].Path
		task.Preset = presetByExt(task.Source)
		if task.Preset == "" {
			return c.String(http.StatusBadRequest, "No preset")
		}
		task.Target = fileutils.AddSlash(srvCtx.Config.TransWork) + uuid.NewV4().String() + ".mp4"
		task.Ctx = r
		if !srvCtx.Trans.Transcode(task) {
			return c.String(http.StatusBadRequest, "Cannot start transcoding")
		}
	} else {
		return c.NoContent(http.StatusNotFound)
	}
	return c.NoContent(http.StatusOK)
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

	if srvCtx.Update != nil {
		srvCtx.Update <- r.Path
	}
	return c.NoContent(http.StatusOK)
}

// GET /api/v1/catalog
func getCatalog(c echo.Context) (err error) {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")

	buf := new(bytes.Buffer)
	files := getfs().GetAll()
	for _, fl := range files {
		storages := make([]string, 0, len(fl))
		for _, fr := range fl {
			if fr.Device == nil {
				log.Println("Wrong device:", fr.Path)
			} else {
				storages = append(storages, fr.Device.Id)
			}
		}
		storagesJson, _ := json.Marshal(storages)
		fmt.Fprintf(buf, "%s,%s\n", fl[0].Sha1, storagesJson)
	}

	return c.Blob(http.StatusOK, "text/plain", buf.Bytes())
}

// GET /api/v1/storages
func getStorages(c echo.Context) (err error) {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")

	ll := make([]fileindex.Storage, 0, 100)
	storages.Range(func(key, value interface{}) bool {
		st := value.(*fileindex.Storage)
		ll = append(ll, *st)
		return true
	})
	return c.JSON(http.StatusOK, ll)
}

func webServer(ctx ServerCtx) {
	srvCtx = ctx

	e := echo.New()
	e.HideBanner = true

	e.GET("/", getHello)
	e.GET("/get/:sha1/:name", getFile)

	e.GET("/api/v1/catalog", getCatalog)
	e.POST("/api/v1/get", postRegFile)
	e.GET("/api/v1/storages", getStorages)
	e.POST("/api/v1/transcode", postTranscode)
	e.POST("/api/v1/update", postUpdate)

	e.Logger.Fatal(e.Start(srvCtx.Config.Listen))
}

func pathTranslate(path string) string {
	path = strings.Replace(path, "\\", "/", -1)
	if x := strings.Index(path, "/Archive/"); x >= 0 {
		path = srvCtx.Config.BasePathArchive + path[x:]
	} else if x := strings.Index(path, "/Archive_PN/"); x >= 0 {
		path = srvCtx.Config.BasePathArchive + path[x:]
	} else if x := strings.Index(path, "/__BACKUP/"); x >= 0 {
		path = srvCtx.Config.BasePathOriginal + path[x:]
	} else {
		path = ""
	}
	return path
}

func updateServer(ctx UpdateCtx) {
	tick := time.Tick(ctx.Config.Reload)
	for {
		select {
		case <-tick:
			if ctx.Index.IsModified() {
				ctx.Index.Load()
			}
		case path := <-ctx.Update:
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

func transcodeResult(tr Transcoder) {
	for {
		r := tr.Result()
		if r.Err == nil {
			handleResult(r.Task)
		} else {
			log.Println("Transcode:", r.Task.Source)
			log.Println("To:", r.Task.Target)
			log.Println("Preset:", r.Task.Preset)
			log.Println(string(r.Out))
		}
		os.Remove(r.Task.Target)
	}
}

func handleResult(t TranscodeTask) {
	req, ok := t.Ctx.(*TranscodeReq)
	if !ok {
		log.Println("Wrong transcoding result")
		return
	}

	sum, size, stat, err := fileutils.SHA1_File(t.Target)
	if err != nil {
		log.Println(err)
		return
	}

	tgtPath := path.Dir(t.Target) + "/" + req.SHA1 + "_" + hex.EncodeToString(sum) + ".mp4"
	err = os.Rename(t.Target, tgtPath)
	if err != nil {
		log.Println(err)
		return
	}

	srcBase := path.Base(t.Source)
	destBase := srcBase[0:len(srcBase)-len(path.Ext(srcBase))] + ".mp4"

	destPath := srvCtx.Config.TransDest + "/" + destBase
	err = os.Link(tgtPath, destPath)
	if err != nil {
		log.Println(err)
		return
	}

	m := map[string]interface{}{
		"original_sha1": req.SHA1,
		"sha1":          hex.EncodeToString(sum),
		"file_name":     destBase,
		"size":          size,
		"created_at":    stat.ModTime().Unix(),
		"station":       "files.kabbalahmedia.info",
		"user":          "operator@dev.com",
	}
	sendNotify(m)
}

func sendNotify(m map[string]interface{}) {
	mJson, _ := json.Marshal(m)
	contentReader := bytes.NewReader(mJson)
	req, _ := http.NewRequest("POST", "http://app.mdb.bbdomain.org/operations/transcode", contentReader)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Println("Notify error:", resp.StatusCode, m["file_name"])
		}
	} else {
		log.Println(err)
	}
}
