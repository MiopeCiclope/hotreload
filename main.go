package main

import (
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
func runBuild() {
	app := "run build -- products"
	command := strings.Split(app, " ")
	cmd := exec.Command("npm", command...)
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

func main() {
	watcher, err := fsnotify.NewWatcher()
	check(err)

	// Create Watchers
	toWatch := Path
	watchRecursive(toWatch, watcher)
	fmt.Println("watching dir: ", toWatch)

	//Flag to avoid trigger of multiple builds on branch checkout
	isBuilding := false

	// Notify when a change happens
	for {
		select {
		case event := <-watcher.Events:
			if !isBuilding && event.Op != fsnotify.Chmod {
				isBuilding = true
				fmt.Println("Build on:", event.String())
				// Run build command
				runBuild()
				isBuilding = false
			} else {
				fmt.Println("No build trigger", event.String())
			}
		case err, ok := <-watcher.Errors:
			fmt.Println(ok, err)
		}
	}
}