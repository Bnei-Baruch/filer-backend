package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"kbb1.com/fileindex"

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
		List    IndexList
		FS      fileindex.FastSearch
		Path    string
		Pattern string
	}

	ServerConf struct {
		Listen  string
		BaseURL string
		Index   *IndexMain
		Update  chan string
	}

	UpdateConf struct {
		Index  *IndexMain
		Reload time.Duration
		Update chan string
	}
)

const (
	localConf  = ".config/filer_storage.conf"
	globalConf = "/etc/filer_storage.conf"
)

func signalHandler() chan os.Signal {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	return signalChan
}

func configLoad() *toml.Tree {
	home := os.Getenv("HOME")
	config, err := toml.LoadFile(home + "/" + localConf)
	if err != nil {
		config, err = toml.LoadFile(globalConf)
		if err != nil {
			log.Fatalln("Error: ", err)
		}
	}
	return config
}

func main() {
	signalChan := signalHandler()

	config := configLoad()
	baseurl := config.Get("server.baseurl").(string)
	listen := config.Get("server.listen").(string)
	reload := config.GetDefault("server.reload", time.Duration(10)).(time.Duration) * time.Second
	path := config.Get("index.dir").(string)
	pattern := config.Get("index.files").(string)
	update := make(chan string, 100)

	index := NewIndex(path, pattern)
	index.Load()

	go webServer(ServerConf{Listen: listen, BaseURL: baseurl, Index: index, Update: update})
	go updateServer(UpdateConf{Index: index, Reload: reload, Update: update})
	_ = <-signalChan
}
