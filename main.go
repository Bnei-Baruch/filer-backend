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
	"kbb1.com/transcode"

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
		BaseURL          string // base URL of the secure file access
		GetFileExpire    time.Duration
		Listen           string
		NotifyStation    string // notify station
		NotifyUser       string // notify user
		TransDest        string // target folder for transcoded files
		TransNotify      string // notify MDB app
		TransWork        string // working folder for transcoder
	}

	TranscoderConf struct {
		Concurrency int // max number of concurrent transcoding processes
	}

	ServerCtx struct {
		Config *ServerConf
		Index  *IndexMain
		Update chan string
		Trans  transcode.Transcoder
	}

	UpdateConf struct {
		BaseDir string
		Reload  time.Duration // rescan interval of the index folder
	}

	UpdateCtx struct {
		Config *UpdateConf
		Index  *IndexMain
		Update chan string
	}

	LocationConf struct {
		Access   string
		Country  string
		Hostname string
		Name     string
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
	Location   LocationConf
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
	conf.Server.GetFileExpire = time.Duration(config.GetDefault("server.getfileexpire", int64(7200)).(int64)) * time.Second

	conf.Server.TransNotify = config.GetDefault("mdbapp.api", "").(string)
	conf.Server.NotifyStation = config.GetDefault("mdbapp.station", "").(string)
	conf.Server.NotifyUser = config.GetDefault("mdbapp.user", "").(string)

	conf.Location.Access = config.GetDefault("location.access", "local").(string)
	conf.Location.Country = config.GetDefault("location.country", "unknown").(string)
	conf.Location.Name = config.GetDefault("location.name", "unknown").(string)
	conf.Location.Hostname = fileutils.BaseHostName()

	conf.Update.BaseDir = config.GetDefault("update.basedir", "/").(string)
	conf.Update.Reload = time.Duration(config.GetDefault("update.reload", int64(10)).(int64)) * time.Second

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

	tr := transcode.NewMultiTranscoder(conf.Transcoder.Concurrency)

	go webServer(ServerCtx{Config: &conf.Server, Index: index, Update: update, Trans: tr})
	go updateServer(UpdateCtx{Config: &conf.Update, Index: index, Update: update})
	go transcodeResult(tr)

	if config.GetDefault("server.stoponupdate", false).(bool) == true {
		go stoponupdate(signalChan)
	}

	_ = <-signalChan
}
