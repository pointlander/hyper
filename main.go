// Copyright 2024 The Hyper Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.bug.st/serial"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	running := true
	go func() {
		<-c
		running = false
	}()

	process := func(device string, output chan uint64) {
		options := &serial.Mode{
			BaudRate: 115200,
		}
		port, err := serial.Open(device, options)
		if err != nil {
			panic(err)
		}

		_, err = port.Write([]byte("<HEARTBEAT1>>"))
		if err != nil {
			panic(err)
		}

		for running {
			data := make([]byte, 2)
			_, err = port.Read(data)
			if err != nil {
				panic(err)
			}
			output <- (uint64(data[0]) << 8) | uint64(data[1])
		}

		_, err = port.Write([]byte("<HEARTBEAT0>>"))
		if err != nil {
			panic(err)
		}

		err = port.Close()
		if err != nil {
			panic(err)
		}

		close(output)
	}

	output1 := make(chan uint64, 8)
	event1 := time.Time{}
	go process("/dev/ttyUSB0", output1)
	output2 := make(chan uint64, 8)
	event2 := time.Time{}
	go process("/dev/ttyUSB1", output2)
	for {
		select {
		case out, ok := <-output1:
			if !ok {
				output1 = nil
				break
			}
			fmt.Println("chan0", out)
			if out > 0 {
				event1 = time.Now()
				diff := event1.Sub(event2)
				if diff < 0 {
					diff = -diff
				}
				if diff < time.Second {
					fmt.Println("event")
				}
			}
		case out, ok := <-output2:
			if !ok {
				output2 = nil
				break
			}
			fmt.Println("chan1", out)
			if out > 0 {
				event2 = time.Now()
				diff := event1.Sub(event2)
				if diff < 0 {
					diff = -diff
				}
				if diff < time.Second {
					fmt.Println("event")
				}
			}
		}
		if output1 == nil && output2 == nil {
			break
		}
	}
}
