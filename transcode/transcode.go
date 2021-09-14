package transcode

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/Bnei-Baruch/filer-backend/fileutils"
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
		QueueLen() int
	}

	MultiTranscoder struct {
		qt chan TranscodeTask
		qr chan TranscodeResult
	}
)

var (
	optargs string = "-hide_banner -nostats -loglevel error -threads 1"
)

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

func transcodeLog(start, finish time.Time, r *TranscodeResult) {
	if r.Err == nil {
		log.Println("Transcode:", finish.Sub(start)/time.Millisecond*time.Millisecond, fileutils.FileSize(r.Task.Source), r.Task.Source)
	}
}

func transcodeRun(qt <-chan TranscodeTask, qr chan<- TranscodeResult) {
	go func() {
		for t := range qt {
			start := time.Now()
			r := TranscodeResult{Task: t}
			r.Err, r.Out = transcodeFile(t.Preset, t.Source, t.Target)
			transcodeLog(start, time.Now(), &r)
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

func (tr *MultiTranscoder) QueueLen() int {
	return len(tr.qt)
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
