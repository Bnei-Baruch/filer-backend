package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kbb1.com/fileindex"
	"kbb1.com/fileutils"
	"kbb1.com/transcode"

	"github.com/labstack/echo"
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

	ShowFormatReq struct {
		SHA1 string `json:"sha1" form:"sha1"`
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

	preset1 string = "-c:v libx264 -profile:v main -preset fast -b:v 128k -c:a libfdk_aac -b:a 48k"
	preset2 string = "-c:v libx264 -profile:v main -preset fast -b:v 256k -c:a libfdk_aac -b:a 48k"
)

func presetByExt(src string) (preset string) {
	ext := filepath.Ext(src)
	switch ext {
	case ".wmv", ".WMV":
		preset = preset1
	case ".flv", ".FLV":
		preset = preset2
	default:
		preset = ""
	}
	return
}

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

func webServer(ctx ServerCtx) {
	srvCtx = ctx

	e := echo.New()
	e.HideBanner = true

	e.GET("/", getHello)
	e.GET("/get/:sha1/:name", getFile)

	e.GET("/api/v1/catalog", getCatalog)
	e.POST("/api/v1/get", postRegFile)
	e.GET("/api/v1/storages", getStorages)
	e.POST("/api/v1/showformat", postShowFormat)
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
			if !strings.HasPrefix(pathtr, ctx.Config.BaseDir) {
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

func transcodeResult(tr transcode.Transcoder) {
	for {
		r := tr.Result()
		if r.Err == nil {
			handleResult(r.Task)
		} else {
			req, ok := r.Task.Ctx.(*TranscodeReq)
			if ok {
				sendError(req.SHA1, string(r.Out))
			}

			log.Println("Transcode:", r.Task.Source)
			log.Println("To:", r.Task.Target)
			log.Println("Preset:", r.Task.Preset)
			log.Println(string(r.Out))
		}
		os.Remove(r.Task.Target)
	}
}

func handleResult(t transcode.TranscodeTask) {
	req, ok := t.Ctx.(*TranscodeReq)
	if !ok {
		log.Println("Wrong transcoding result")
		return
	}

	sum, size, stat, err := fileutils.SHA1_File(t.Target)
	if err != nil {
		sendError(req.SHA1, err.Error())
		log.Println(err)
		return
	}

	// finalize the name of the transcoded file in the working folder
	tgtPath := path.Dir(t.Target) + "/" + req.SHA1 + "_" + hex.EncodeToString(sum) + ".mp4"
	err = os.Rename(t.Target, tgtPath)
	if err != nil {
		log.Println(err)
		sendError(req.SHA1, err.Error())
		return
	}

	// make a hard link from the working folder to the destination folder
	srcBase := path.Base(t.Source)
	destBase := srcBase[0:len(srcBase)-len(path.Ext(srcBase))] + ".mp4"

	destPath := srvCtx.Config.TransDest + destBase
	err = os.Link(tgtPath, destPath)
	if err != nil {
		log.Println(err)
		sendError(req.SHA1, err.Error())
		return
	}

	// send update notify to indexer
	srvCtx.Update <- destPath

	// send the transcoding result to MDB application
	if len(srvCtx.Config.TransNotify) > 0 {
		m := map[string]interface{}{
			"original_sha1": req.SHA1,
			"sha1":          hex.EncodeToString(sum),
			"file_name":     destBase,
			"size":          size,
			"created_at":    stat.ModTime().Unix(),
			"station":       srvCtx.Config.NotifyStation,
			"user":          srvCtx.Config.NotifyUser,
		}
		sendNotify(srvCtx.Config.TransNotify, m)
	}
}

func sendError(sha1 string, msg string) {
	m := map[string]interface{}{
		"original_sha1": sha1,
		"message":       msg,
		"station":       srvCtx.Config.NotifyStation,
		"user":          srvCtx.Config.NotifyUser,
	}
	sendNotify(srvCtx.Config.TransNotify, m)
}

func sendNotify(api string, m map[string]interface{}) {
	mJson, _ := json.Marshal(m)
	contentReader := bytes.NewReader(mJson)
	req, _ := http.NewRequest("POST", api, contentReader)
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
