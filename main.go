package main

import (
	"context"
	"fmt"
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
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(stdout))
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

// func buildPrjects(file chan string) {
// 	isRunning := false
// 	for {
// 		select {
// 		case fileName := <-file:
// 			fmt.Println(isRunning)

// 			if !isRunning {
// 				isRunning = true
// 				time.Sleep(5 * time.Second)
// 				fmt.Println("Build on:", fileName)
// 				// Run build command
// 				// runBuild()
// 				isRunning = false
// 			} else {
// 				fmt.Println("Build already on")
// 			}
// 		}
// 	}
// }

func main() {
	watcher, err := fsnotify.NewWatcher()
	check(err)

	// Create Watchers
	toWatch := Path
	watchRecursive(toWatch, watcher)
	fmt.Println("watching dir: ", toWatch)

	cancellationToken := make(chan context.CancelFunc)
	storeCancellation := byPass()
	// Notify when a change happens
	for {
		select {
		case event := <-watcher.Events:
			if event.Op != fsnotify.Chmod {
				fmt.Println("Start build")
				ctx, cancel := context.WithCancel(context.Background())
				storeCancellation(cancel, cancellationToken)
				fmt.Println("Build Trigger:", event.String())
				// Run build command
				runBuild(ctx)
			}
		case cancellationFunction := <-cancellationToken:
			fmt.Println("Cancel trigger")
			cancellationFunction()
		case err, ok := <-watcher.Errors:
			fmt.Println(ok, err)
		}
	}
}

func byPass() func(context.CancelFunc, chan context.CancelFunc) {
	var previousFunction context.CancelFunc
	return func(cancelFunction context.CancelFunc, channel chan context.CancelFunc) {
		if previousFunction != nil {
			channel <- previousFunction
		}
		previousFunction = cancelFunction
	}
}
