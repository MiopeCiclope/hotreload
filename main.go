package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

const (
	Path = "/Users/romulotone/projects/eti-web/"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Find all folders inside root to watch
func watchRecursive(path string, watcher *fsnotify.Watcher) error {
	err := filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if isBlackListed(fi.Name()) {
				return filepath.SkipDir
			}

			if err = watcher.Add(walkPath); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// Ignore folders changed on build
func isBlackListed(folderName string) bool {
	blackList := []string{"dist", "node_modules", "build", ".git"}

	for _, name := range blackList {
		if strings.Contains(folderName, name) {
			return true
		}
	}
	return false
}

func main() {
	watcher, err := fsnotify.NewWatcher()
	check(err)

	// Create Watchers
	toWatch := Path
	watchRecursive(toWatch, watcher)
	threadCounter := counter()
	fmt.Println("watching dir: ", toWatch)
	quit := make(chan bool)
	// Notify when a change happens
	for {
		select {
		case event := <-watcher.Events:
			if event.Op != fsnotify.Chmod {
				// fmt.Println("Build Trigger:", event.String())
				quit <- true
				// Run build command
				daemon := getInstance()
				go daemon.Start(threadCounter, quit)
			}
		case err, ok := <-watcher.Errors:
			fmt.Println(ok, err)
		}
	}
}

func counter() func() int {
	sum := 0
	return func() int {
		sum++
		return sum
	}
}

// var lock = &sync.Mutex{}

var singleInstance *Daemon

func getInstance() *Daemon {
	// lock.Lock()
	if singleInstance == nil {
		fmt.Println("Creating single instance now.")
		singleInstance = &Daemon{}
	} else {
		singleInstance.Cancel()
		fmt.Println("Single instance already created.")
	}
	// lock.Unlock()

	return singleInstance
}

type Daemon struct {
	cmdErr error
	cancel func()
}

func (d *Daemon) Start(getId func() int, quit chan bool) {
	tId := getId()
	fmt.Println("Thread-", tId, " -> start")

	select {
	case <-quit:
		fmt.Println("Thread-", tId, " Teje morto")

		return
	default:

		ctx, cancel := context.WithCancel(context.Background())
		d.cancel = cancel

		app := "run build -- products"
		command := strings.Split(app, " ")
		cmd := exec.CommandContext(ctx, "npm", command...)

		toExecute := filepath.Dir(Path)
		cmd.Dir = toExecute

		outR, outW := io.Pipe()
		cmd.Stdout = io.MultiWriter(outW, os.Stdout)
		cmd.Stderr = os.Stderr
		lines := make(chan string)

		go func(threadId int) {
			for line := range lines {
				fmt.Println("Thread-", threadId, " -> ", line)
			}
		}(tId)

		go func(threadId int) {
			scanner := bufio.NewScanner(outR)
			for scanner.Scan() {
				lines <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				fmt.Println("Thread-", threadId, " -> ", "scanner: ", err)
			}
			close(lines)
		}(tId)

		err := cmd.Start()
		if err != nil {
			fmt.Println("Thread-", tId, " -> ", "Fatal", err)
		}

		go func(threadId int) {
			err := cmd.Wait()
			fmt.Println("Thread-", threadId, " -> ", "command exited; error is:", err)
			_ = outW.Close() // TODO: handle error from Close(); log it maybe.
			d.cmdErr = err
		}(tId)
	}
}

func (d *Daemon) Cancel() {
	d.cancel()
}

func (d *Daemon) CmdErr() error {
	return d.cmdErr
}
