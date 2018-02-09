package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
	"kbb1.com/fileindex"
	"kbb1.com/fileutils"
	"kbb1.com/transcode"
)

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

// POST /api/v1/showformat
func postShowFormat(c echo.Context) (err error) {
	c.Response().Header().Set(echo.HeaderAccessControlAllowOrigin, "*")

	r := new(ShowFormatReq)
	if err = c.Bind(r); err != nil {
		return c.String(http.StatusBadRequest, "Wrong parameters")
	}
	if r.SHA1 == "" {
		return c.String(http.StatusBadRequest, "Wrong parameters")
	}

	if fl, ok := search(r.SHA1); ok {
		err, out := transcode.ShowFormat(fl[0].Path)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		if !json.Valid(out) {
			return c.String(http.StatusBadRequest, "Invalid result")
		}
		return c.JSONBlob(http.StatusOK, out)
	}
	return c.NoContent(http.StatusNotFound)
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
		err, probe := transcode.Probe(fl[0].Path)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		var task transcode.TranscodeTask
		task.Source = fl[0].Path
		task.Preset = preset(probe)
		if task.Preset == "" {
			return c.String(http.StatusBadRequest, "No preset")
		}
		task.Target = fileutils.AddSlash(srvCtx.Config.TransWork) + uuid.NewV4().String() + ".mp4"
		task.Ctx = r
		if !srvCtx.Trans.Transcode(task) {
			return c.String(http.StatusBadRequest, "Cannot start transcoding")
		}
		return c.NoContent(http.StatusOK)
	}
	return c.NoContent(http.StatusNotFound)
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
