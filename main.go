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

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/generators"
	"github.com/gopxl/beep/speaker"
	"go.bug.st/serial"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	running := true
	go func() {
		<-c
		running = false
	}()

	sr := beep.SampleRate(48000)
	speaker.Init(sr, 4800)

	sine, err := generators.SineTone(sr, 17000.0)
	if err != nil {
		panic(err)
	}

	beep := func() {
		two := sr.N(250 * time.Millisecond)

		ch := make(chan struct{})
		sounds := []beep.Streamer{
			beep.Take(two, sine),
			beep.Callback(func() {
				ch <- struct{}{}
			}),
		}
		speaker.Play(beep.Seq(sounds...))

		<-ch
	}

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
	event1 := time.Now()
	go process("/dev/ttyUSB0", output1)
	output2 := make(chan uint64, 8)
	event2 := time.Now()
	go process("/dev/ttyUSB1", output2)
	last := time.Now()
	var values, between plotter.Values
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
					go beep()
					values = append(values, float64(event1.Sub(last)))
					last = event1
				}
				between = append(between, float64(diff))
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
					go beep()
					values = append(values, float64(event2.Sub(last)))
					last = event2
				}
				between = append(between, float64(diff))
			}
		}
		if output1 == nil && output2 == nil {
			break
		}
	}

	p := plot.New()
	p.Title.Text = "histogram plot"

	hist, err := plotter.NewHist(values, 40)
	if err != nil {
		panic(err)
	}
	p.Add(hist)

	if err := p.Save(8*vg.Inch, 8*vg.Inch, "hist.png"); err != nil {
		panic(err)
	}

	p = plot.New()
	p.Title.Text = "histogram between plot"

	hist, err = plotter.NewHist(between, 40)
	if err != nil {
		panic(err)
	}
	p.Add(hist)

	if err := p.Save(8*vg.Inch, 8*vg.Inch, "between.png"); err != nil {
		panic(err)
	}
}
