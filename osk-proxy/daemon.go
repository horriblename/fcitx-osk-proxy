package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"path"
	"time"
)

var gOpenOSK = []string{"wf-osk"}
var gCloseOSK = []string{"pkill", "wf-osk"}

const gDebounce = 200 * time.Millisecond

type Msg string

const (
	gImeEnter Msg = "ImeEnter"
	gImeLeave Msg = "ImeLeave"
)

func dbg(msg ...any) { log.Print(msg...) }

func assertOk(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func socketPath() string {
	tempDir := os.Getenv("XDG_RUNTIME_DIR")
	if tempDir == "" {
		tempDir = "/tmp"
	}
	tempDir = path.Join(tempDir, "fcitx-osk-proxy")

	waylandSession := os.Getenv("WAYLAND_DISPLAY")
	if waylandSession == "" {
		log.Fatal("no $WAYLAND_DISPLAY")
	}

	assertOk(os.MkdirAll(tempDir, 0o700))

	return path.Join(tempDir, waylandSession+".sock")
}

// kill the loop by closing r, probably not thread safe but who cares
func readLoop(r *bufio.Scanner) <-chan Msg {
	msg := make(chan Msg)
	go func() {
		defer close(msg)

		for r.Scan() {
			text := r.Text()
			switch Msg(text) {
			case gImeEnter, gImeLeave:
				dbg("sending msg", text)
				msg <- Msg(text)
				dbg("sent", text)
			default:
				log.Print("[WARN] invalid msg: ", text)
			}
		}

		if r.Err() != nil {
			log.Print("readLoop: ", r.Err())
		}
	}()
	return msg
}

type program struct {
	oskActive bool
}

func openOSK() error {
	dbg("opening OSK")
	cmd := exec.Command(gOpenOSK[0], gOpenOSK[1:]...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func closeOSK() error {
	dbg("closing OSK")
	cmd := exec.Command(gCloseOSK[0], gCloseOSK[1:]...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func (p *program) debounceLeave(timer *time.Timer, msgs <-chan Msg) {
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(gDebounce)

loop:
	for {
		select {
		case msg, more := <-msgs:
			if !more {
				log.Print("[INFO] msgs closed")
				break loop
			}

			switch msg {
			case gImeLeave:
				continue
			case gImeEnter:
				if !timer.Stop() {
					<-timer.C
				}
				dbg("debounce stopped by enter event")
				return
			default:
				panic("unreachable")
			}

		case <-timer.C:
			err := closeOSK()
			if err != nil {
				log.Print("[ERR] closing osk: ", err)
			}
			p.oskActive = false
			return
		}
	}
}

func main() {
	msgs := readLoop(bufio.NewScanner(os.Stdin))
	program := program{}
	timer := time.NewTimer(0)

	for msg := range msgs {
		dbg("got msg: ", msg)
		switch msg {
		case gImeEnter:
			if program.oskActive {
				dbg("OSK already active")
				continue
			}

			err := openOSK()
			if err != nil {
				log.Print("[ERR] opening OSK: ", err)
			}
			program.oskActive = true

		case gImeLeave:
			if !program.oskActive {
				dbg("OSK already closed")
				continue
			}
			program.debounceLeave(timer, msgs)
		}
	}
}
