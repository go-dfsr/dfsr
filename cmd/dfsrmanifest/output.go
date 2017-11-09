package main

import "fmt"

// Output is an output channel
type Output chan string

// Printf calls fmt.Sprintf and sends the result to the output channel
func (o Output) Printf(format string, v ...interface{}) {
	o <- fmt.Sprintf(format, v...)
}

// Println calls fmt.Sprintln and sends the result to the output channel
func (o Output) Println(a ...interface{}) {
	o <- fmt.Sprintln(a...)
}
