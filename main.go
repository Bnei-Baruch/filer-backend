package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"kbb1.com/fileindex"
	"kbb1.com/fileutils"

	"github.com/pelletier/go-toml"
)

type (
	IndexFile struct {
		Path  string
		Mtime int64
		Files fileindex.FileList
	}

	IndexList []IndexFile

	IndexMain struct {
		sync.Mutex
		List IndexList
		fs   *fileindex.FastSearch
		Path string
	}

	ServerConf struct {
		BasePathArchive  string
		BasePathOriginal string
		BaseURL          string // Base URL of the secure file access
		Listen           string
		TransDest        string // Traget folder for transcoded files
		TransWork        string // Working folder for transcoder
	}

	TranscoderConf struct {
		Concurrency int // Max number of concurrent transcoding processes
	}

	ServerCtx struct {
		Config *ServerConf
		Index  *IndexMain
		Update chan string
		Trans  Transcoder
	}

	UpdateConf struct {
		Reload time.Duration // Rescan interval of the index folder
	}

	UpdateCtx struct {
		Config *UpdateConf
		Index  *IndexMain
		Update chan string
	}
)

const (
	localConf  = ".config/filer_storage.conf"
	globalConf = "/etc/filer_storage.conf"
)

func configLoad() *toml.Tree {
	home := os.Getenv("HOME")
	config, err := toml.LoadFile(home + "/" + localConf)
	if err != nil {
		if err == os.ErrNotExist {
			config, err = toml.LoadFile(globalConf)
		}
		if err != nil {
			log.Fatalln("Load config file: ", err)
		}
	}
	return config
}

func signalHandler() chan os.Signal {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	return signalChan
}

// Stop the program in case the executable file is updated
func stoponupdate(ch chan os.Signal) {
	prog, _ := filepath.Abs(os.Args[0])
	stat, _ := os.Stat(prog)

	for {
		time.Sleep(time.Second * 2)
		if s, err := os.Stat(prog); err == nil {
			// skip if it's being updated now
			if time.Now().Sub(s.ModTime()) < time.Second*2 {
				continue
			}
			if s.ModTime() != stat.ModTime() {
				log.Println("Stop on update")
				ch <- syscall.SIGQUIT
				break
			}
		}
	}
}

var conf struct {
	Server     ServerConf
	Transcoder TranscoderConf
	Update     UpdateConf
}

func main() {
	signalChan := signalHandler()

	config := configLoad()
	conf.Server.BasePathArchive = config.Get("server.basepath.Archive").(string)
	conf.Server.BasePathOriginal = config.Get("server.basepath.Original").(string)
	conf.Server.BaseURL = config.Get("server.baseurl").(string)
	conf.Server.Listen = config.Get("server.listen").(string)
	conf.Update.Reload = config.GetDefault("server.reload", time.Duration(10)).(time.Duration) * time.Second

	conf.Transcoder.Concurrency = int(config.GetDefault("transcoder.concurrency", int64(0)).(int64))
	if conf.Transcoder.Concurrency > 0 {
		conf.Server.TransDest = fileutils.AddSlash(config.Get("server.transdest").(string))
		conf.Server.TransWork = fileutils.AddSlash(config.Get("server.transwork").(string))
	}

	log.SetOutput(fileutils.NewLogWriter(fileutils.LogCtx{
		Path: config.GetDefault("server.log", "").(string),
	}))

	InitStorages()

	index := NewIndex(config.Get("index.dir").(string))
	index.Load()
	update := make(chan string, 100)

	tr := NewMultiTranscoder(conf.Transcoder.Concurrency)

	go webServer(ServerCtx{Config: &conf.Server, Index: index, Update: update, Trans: tr})
	go updateServer(UpdateCtx{Config: &conf.Update, Index: index, Update: update})
	go transcodeResult(tr)

	if config.GetDefault("server.stoponupdate", false).(bool) == true {
		go stoponupdate(signalChan)
	}

	_ = <-signalChan
}
