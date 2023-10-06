package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
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
		fmt.Println(line)
	}
}

func (b Builder) cmdWriter() {
	scanner := bufio.NewScanner(b.Reader)
	for scanner.Scan() {
		b.IoChannel <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	close(b.IoChannel)
}

func printTitleWithBorders(title string, lineLength int) string {
	titleLength := len(title)
	totalSpaces := lineLength - titleLength
	leftSpaces := totalSpaces / 2
	rightSpaces := totalSpaces - leftSpaces

	formattedText := strings.Repeat("=", lineLength) + "\n" +
		strings.Repeat(" ", leftSpaces) + title + strings.Repeat(" ", rightSpaces) + "\n" +
		strings.Repeat("=", lineLength)

	return formattedText
}

func (b Builder) cmdExit() {
	err := b.Cmd.Wait()

	if err != nil {
		fmt.Println("Build Stopped:", err)
	} else {
		fmt.Println(printTitleWithBorders("Build Done!", 40))
	}

	b.Writer.Close()
}

func (b Builder) run() {
	go b.cmdReader()
	go b.cmdWriter()
	b.Cmd.Start()

	go b.cmdExit()
}
