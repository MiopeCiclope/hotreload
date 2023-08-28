package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

type Builder struct {
	ThreadId  int
	Cmd       *exec.Cmd
	Reader    *io.PipeReader
	Writer    *io.PipeWriter
	IoChannel chan string
}

func (b Builder) cmdReader() {
	for line := range b.IoChannel {
		fmt.Println("Thread-", b.ThreadId, " -> ", line)
	}
}

func (b Builder) cmdWriter() {
	scanner := bufio.NewScanner(b.Reader)
	for scanner.Scan() {
		b.IoChannel <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Thread-", b.ThreadId, " -> ", "scanner: ", err)
	}
	close(b.IoChannel)
}

func (b Builder) cmdRun() {
	err := b.Cmd.Start()
	if err != nil {
		fmt.Println("Thread-", b.ThreadId, " -> ", "Fatal", err)
	}
}

func (b Builder) cmdExit() {
	err := b.Cmd.Wait()
	fmt.Println("Thread-", b.ThreadId, " -> ", "command exited; error is:", err)
	b.Writer.Close()
}

func (b Builder) run() {
	fmt.Println("Thread-", b.ThreadId, " -> start")
	go b.cmdReader()
	go b.cmdWriter()
	b.cmdRun()

	go b.cmdExit()
}
