package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	TranscodeTask struct {
		Ctx    interface{}
		Preset string
		Source string
		Target string
	}

	TranscodeResult struct {
		Task TranscodeTask
		Err  error
		Out  []byte
	}

	Transcoder interface {
		Transcode(task TranscodeTask) bool
		Result() TranscodeResult
	}

	MultiTranscoder struct {
		qt chan TranscodeTask
		qr chan TranscodeResult
	}
)

var (
	optargs string = "-hide_banner -nostats -loglevel error -threads 1"
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

func transcodeFile(preset, srcpath, dstpath string) (error, []byte) {
	params := strings.Fields(optargs)
	params = append(params, "-i", srcpath)
	params = append(params, strings.Fields(preset)...)
	params = append(params, dstpath)

	cmd := exec.Command("ffmpeg", params...)

	var out bytes.Buffer
	cmd.Stderr = &out

	err := cmd.Start()
	if err != nil {
		return err, nil
	}
	err = cmd.Wait()
	return err, out.Bytes()
}

func transcodeRun(qt <-chan TranscodeTask, qr chan<- TranscodeResult) {
	go func() {
		for t := range qt {
			r := TranscodeResult{Task: t}
			r.Err, r.Out = transcodeFile(t.Preset, t.Source, t.Target)
			qr <- r
		}
	}()
}

func NewMultiTranscoder(concurrency int) *MultiTranscoder {
	mt := &MultiTranscoder{
		qt: make(chan TranscodeTask, 100),
		qr: make(chan TranscodeResult, 100),
	}

	for i := 0; i < concurrency; i++ {
		transcodeRun(mt.qt, mt.qr)
	}

	return mt
}

func (tr *MultiTranscoder) Result() TranscodeResult {
	return <-tr.qr
}

func (tr *MultiTranscoder) Transcode(task TranscodeTask) bool {
	select {
	case tr.qt <- task:
		return true
	default:
	}
	return false
}
