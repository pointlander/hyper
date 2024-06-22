// Copyright 2024 The Hyper Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	options := &serial.Mode{
		BaudRate: 115200,
	}
	port, err := serial.Open("/dev/ttyUSB0", options)
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
		fmt.Println((uint64(data[0]) << 8) | uint64(data[1]))
	}

	_, err = port.Write([]byte("<HEARTBEAT0>>"))
	if err != nil {
		panic(err)
	}

	err = port.Close()
	if err != nil {
		panic(err)
	}
}
