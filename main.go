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
	"time"

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

	var daemon Daemon

	// Notify when a change happens
	for {
		select {
		case event := <-watcher.Events:
			if event.Op != fsnotify.Chmod {
				// fmt.Println("Build Trigger:", event.String())
				// Run build command
				go daemon.Start()
			}
		case err, ok := <-watcher.Errors:
			fmt.Println(ok, err)
		}
	}
}

type Daemon struct {
	cmdErr error
	cancel func()
}

func (d *Daemon) Start() {
	fmt.Println("Start")
	if d.cancel != nil {
		d.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	app := "run build -- products"
	command := strings.Split(app, " ")
	cmd := exec.CommandContext(ctx, "npm", command...)
	toExecute := filepath.Dir(Path)
	cmd.Dir = toExecute

	// outR, outW := io.Pipe()
	// cmd.Stdout = io.MultiWriter(outW, os.Stdout)
	// cmd.Stderr = os.Stderr
	// lines := make(chan string)
	// go func() {
	// 	for line := range lines {
	// 		fmt.Println(line)
	// 	}
	// }()

	// go func() {
	// 	defer close(lines)
	// 	scanner := bufio.NewScanner(outR)
	// 	for scanner.Scan() {
	// 		lines <- scanner.Text()
	// 	}
	// 	if err := scanner.Err(); err != nil {
	// 		fmt.Println("scanner: ", err)
	// 	}
	// }()
	time.Sleep(500 * time.Millisecond)
	err := cmd.Start()
	if err != nil {
		if cmd.Process != nil {
			fmt.Println("Killing time")
			cmd.Process.Kill()
		}
		fmt.Println("Fatal", err)
	}

	go func() {
		err := cmd.Wait()
		// if cmd.Process != nil {
		// 	cmd.Process.Kill()
		// }
		fmt.Println("command exited; error is:", err)
		// _ = outW.Close() // TODO: handle error from Close(); log it maybe.
		d.cmdErr = err
	}()
}

// Cancel causes the running command to exit preemptively.
// If Cancel is called after the command has already
// exited either naturally or due to a previous Cancel call,
// then Cancel has no effect.
func (d *Daemon) Cancel() {
	d.cancel()
}

// CmdErr returns the error, if any, from the command's exit.
// Only valid after the channel returned by Done() has been closed.
func (d *Daemon) CmdErr() error {
	return d.cmdErr
}
