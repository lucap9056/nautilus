package configwatcher

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"nautilus/internal/core/proxy"
	"nautilus/internal/rtree"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type ConfigWatcher struct {
	configDirectory string
	configFileName  string
	fullConfigPath  string
	ntlcPath        string
	manager         *proxy.Manager
	isSource        bool
	fw              *fsnotify.Watcher
}

func NewConfigWatcher(configPath, ntlcPath string, manager *proxy.Manager) (*ConfigWatcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		absPath = configPath
	}

	return &ConfigWatcher{
		configDirectory: filepath.Dir(configPath),
		configFileName:  filepath.Base(configPath),
		fullConfigPath:  absPath,
		ntlcPath:        ntlcPath,
		manager:         manager,
		isSource:        !strings.HasSuffix(configPath, ".ntl"),
		fw:              fw,
	}, nil
}

func (cw *ConfigWatcher) LoadInitial() error {
	var tree *rtree.RouteTree
	var err error

	if cw.isSource {
		tree, err = cw.compileAndLoad()
	} else {
		tree, err = cw.loadStatic()
	}

	if err != nil {
		return err
	}

	cw.manager.UpdateTree(tree)
	return nil
}

func (cw *ConfigWatcher) Start() error {
	if err := cw.fw.Add(cw.configDirectory); err != nil {
		return fmt.Errorf("failed to watch config file: %v", err)
	}

	go cw.listen()
	log.Printf("Config Watcher started for: %s/%s", cw.configDirectory, cw.configFileName)
	return nil
}

func (cw *ConfigWatcher) listen() {
	var timer *time.Timer

	for {
		select {
		case event, ok := <-cw.fw.Events:
			if !ok {
				return
			}

			if filepath.Base(event.Name) != cw.configFileName {
				continue
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				log.Printf("Config file change detected: %v", event.Op)

				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(100*time.Millisecond, func() {
					cw.reload()
				})
			}
		case err, ok := <-cw.fw.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)
		}
	}
}

func (cw *ConfigWatcher) reload() {
	var newTree *rtree.RouteTree
	var err error

	if cw.isSource {
		newTree, err = cw.compileAndLoad()
	} else {
		newTree, err = cw.loadStatic()
	}

	if err != nil {
		log.Printf("Error reloading route table: %v", err)
		return
	}

	cw.manager.UpdateTree(newTree)
	log.Println("Successfully reloaded and updated route table")
}

func (cw *ConfigWatcher) loadStatic() (*rtree.RouteTree, error) {
	file, err := os.Open(cw.fullConfigPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tree rtree.RouteTree
	dec := gob.NewDecoder(file)
	if err := dec.Decode(&tree); err != nil {
		return nil, err
	}
	return &tree, nil
}

func (cw *ConfigWatcher) compileAndLoad() (*rtree.RouteTree, error) {
	cmd := exec.Command(cw.ntlcPath, "-i", cw.fullConfigPath, "-o", "-")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var tree rtree.RouteTree
	dec := gob.NewDecoder(stdout)
	decodeErr := dec.Decode(&tree)

	slurp, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("ntlc failed: %v, stderr: %s", err, string(slurp))
	}

	if decodeErr != nil {
		return nil, fmt.Errorf("decode compiled data failed: %v", decodeErr)
	}

	return &tree, nil
}

func (cw *ConfigWatcher) Close() error {
	return cw.fw.Close()
}
