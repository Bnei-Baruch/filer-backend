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

	Index struct {
		sync.Mutex
		List    IndexList
		SHA1    fileindex.FastSearch
		Path    string
		Pattern string
	}
)

const (
	localConf  = ".config/filer_storage.conf"
	globalConf = "/etc/filer_storage.conf"
)

var (
	index Index
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

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	signalChan := signalHandler()

	config := configLoad()
	listen := config.Get("server.listen").(string)
	reload := config.GetDefault("server.reload", time.Duration(10)).(time.Duration) * time.Second
	path := config.Get("index.dir").(string)
	pattern := config.Get("index.files").(string)

	index = NewIndex(path, pattern)
	index.Load()

	go server(listen)

	go func() {
		c := time.Tick(reload)
		for _ = range c {
			if index.IsModified() {
				log.Println("Reload indexes")
				index.Load()
			}
		}
	}()

	_ = <-signalChan
}
