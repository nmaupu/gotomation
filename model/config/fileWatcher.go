package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/routines"
	"github.com/spf13/viper"
)

// FileWatcher is a watcher for file changes, it loads it using viper (mapstructure)
type FileWatcher interface {
	routines.Runnable
	SetFilename(filename string)
}

type fileWatcher struct {
	*fsnotify.Watcher
	onChangeFuncs []func()
	mutex         sync.Mutex
	started       bool
	filename      string
	getTypeFunc   func() interface{}
	stopChan      chan bool
}

// NewFileWatcher returns a FileWatcher
func NewFileWatcher(filename string, getTypeFunc func() interface{}) FileWatcher {
	l := logging.NewLogger("NewFileWatcher").With().Str("filename", filename).Logger()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		l.Fatal().
			Err(err).
			Msg("Unable to create a FileWatcher")
	}

	return &fileWatcher{
		Watcher:     watcher,
		filename:    filename,
		getTypeFunc: getTypeFunc,
		stopChan:    make(chan bool, 1),
	}
}

// Start starts the watcher
func (w *fileWatcher) Start() error {
	l := logging.NewLogger("FileWatcher.Start").With().Str("filename", w.filename).Logger()
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.started {
		return nil
	}

	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()

		err := w.loadConf()
		if err != nil {
			l.Error().Err(err).Send()
		}

	loop:
		for {
			l.Info().Msg("Looping")
			select {
			case event := <-w.Watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					err := w.loadConf()
					if err != nil {
						l.Error().Err(err).Send()
					}
				}
			case err := <-w.Watcher.Errors:
				l.Error().Err(err).Msg("An error occurred watching file")
			case <-w.stopChan:
				break loop
			}
		}
	}()

	w.started = true
	return w.Watcher.Add(w.filename)
}

// Stop stops the watcher
func (w *fileWatcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.started {
		return
	}

	w.stopChan <- true
	w.Watcher.Close()
	w.started = false

}

func (w *fileWatcher) IsStarted() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.started
}

// GetName returns the name of this runnable object
func (w *fileWatcher) GetName() string {
	return fmt.Sprintf("%s FileWatcher", w.filename)
}

// SetFilename sets the filename to watch
func (w *fileWatcher) SetFilename(filename string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.filename == filename {
		return
	}

	_ = w.Watcher.Remove(w.filename)
	w.filename = filename
	_ = w.Watcher.Add(filename)
	_ = w.loadConf()
}

func (w *fileWatcher) loadConf() error {
	vi := viper.New()
	vi.SetConfigFile(w.filename)

	err := vi.ReadInConfig()
	if err != nil {
		return err
	}

	result := w.getTypeFunc()
	decoderConfigFunc := func(config *mapstructure.DecoderConfig) {
		config.Result = result
		config.DecodeHook = MapstructureDecodeHookFunc()
	}
	return vi.Unmarshal(result, decoderConfigFunc)
}
