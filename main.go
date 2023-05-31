package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
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

// Build command for products package
func runBuild(ctx context.Context) {
	app := "run build -- products"
	command := strings.Split(app, " ")
	cmd := exec.CommandContext(ctx, "npm", command...)
	toExecute := filepath.Dir(Path)
	cmd.Dir = toExecute

	outR, outW := io.Pipe()
	cmd.Stdout = io.MultiWriter(outW, os.Stdout)
	cmd.Stderr = os.Stderr

	lines := make(chan string)
	go func() {
		for line := range lines {
			fmt.Println(line)
		}
	}()

	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(outR)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Println(err)
		}
	}()

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = cmd.Wait()
	_ = outW.Close()
	if err != nil {
		return
	}

	fmt.Println("Kept running")
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
	fmt.Println("watching dir: ", toWatch)

	storeCancellation := byPass()
	// Notify when a change happens
	for {
		select {
		case event := <-watcher.Events:
			if event.Op != fsnotify.Chmod {
				ctx, cancel := context.WithCancel(context.Background())
				storeCancellation(cancel)
				fmt.Println("Build Trigger:", event.String())
				// Run build command
				go runBuild(ctx)
			}
		case err, ok := <-watcher.Errors:
			fmt.Println(ok, err)
		}
	}
}

// On first call saves the cancel that needs to be trigger
// in the next call (no context to close on first, none running)
func byPass() func(context.CancelFunc) {
	var previousFunction context.CancelFunc
	return func(cancelFunction context.CancelFunc) {
		if previousFunction != nil {
			previousFunction()
		}
		previousFunction = cancelFunction
	}
}
