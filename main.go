package main

import (
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
	// Command = "npm run build -- products"
	Command = "npm run build -- --ignore=@eti/client"
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
	blackList := []string{"dist", "node_modules", "build", ".git", "logs"}

	for _, name := range blackList {
		if strings.Contains(folderName, name) {
			return true
		}
	}
	return false
}

func createCommand(threadId int, ctx context.Context, command string, path string) Builder {
	commandSplit := strings.Split(command, " ")
	cmd := exec.CommandContext(ctx, commandSplit[0], commandSplit[1:]...)
	toExecute := filepath.Dir(Path)
	cmd.Dir = toExecute

	outR, outW := io.Pipe()
	cmd.Stdout = io.MultiWriter(outW, os.Stdout)
	cmd.Stderr = os.Stderr
	lines := make(chan string)

	return Builder{
		ThreadId:  threadId,
		Cmd:       cmd,
		Reader:    outR,
		Writer:    outW,
		IoChannel: lines,
	}
}

func main() {
	watcher, err := fsnotify.NewWatcher()
	check(err)

	// Create Watchers
	toWatch := Path
	watchRecursive(toWatch, watcher)
	threadCounter := 0
	fmt.Println("watching dir: ", toWatch)
	var cancelOut context.CancelFunc

	// Notify when a change happens
	for {
		select {
		case event := <-watcher.Events:
			fmt.Println("Event trigger: ", event.Name)
			if event.Op != fsnotify.Chmod {
				fmt.Println("Event should build: ", event.Name)

				threadCounter++
				if cancelOut != nil {
					cancelOut()
				}

				ctx, cancel := context.WithCancel(context.Background())
				cancelOut = cancel
				builder := createCommand(threadCounter, ctx, Command, Path)
				go builder.run()
			}
		case err, ok := <-watcher.Errors:
			fmt.Println(ok, err)
		}
	}
}
