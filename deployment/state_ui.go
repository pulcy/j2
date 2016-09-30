// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// +build !windows

package deployment

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gosuri/uilive"
	"github.com/pulcy/j2/scheduler"
	"github.com/ryanuber/columnize"
)

// ESC is the ASCII code for escape character
const (
	ESC = 27
)

type stateUI struct {
	HeaderSink  chan string
	EventSink   chan scheduler.Event
	MessageSink chan string

	mutex        sync.Mutex
	lastHeader   string
	lastMessage  string
	states       map[string]string
	writer       *uilive.Writer
	stopChan     chan bool
	stopWait     sync.WaitGroup
	verbose      bool
	bypassWriter io.Writer
	autoConfirm  bool
}

func newStateUI(verbose bool) *stateUI {
	s := &stateUI{
		HeaderSink:  make(chan string),
		EventSink:   make(chan scheduler.Event),
		MessageSink: make(chan string),
		states:      make(map[string]string),
		writer:      uilive.New(),
		stopChan:    make(chan bool),
		verbose:     verbose,
		autoConfirm: false,
	}
	s.bypassWriter = s.writer.Bypass()
	//s.writer.Start()
	go s.processSinks()
	return s
}

func (s *stateUI) Close() {
	s.stopWait.Add(1)
	s.stopChan <- true
	s.stopWait.Wait()

	close(s.HeaderSink)
	close(s.EventSink)
	close(s.MessageSink)
	s.writer.Stop()
}

func (s *stateUI) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.lastHeader = ""
	s.states = make(map[string]string)
	s.lastMessage = ""
}

func (s *stateUI) Confirm(question string) error {
	prefix := ""
	for {
		var line string
		s.MessageSink <- fmt.Sprintf("%s%s [yes|no|all]", prefix, question)
		if s.autoConfirm {
			line = "yes"
		} else {
			bufStdin := bufio.NewReader(os.Stdin)
			lineRaw, _, err := bufStdin.ReadLine()
			if err != nil {
				return err
			}
			line = string(lineRaw)
		}
		clearLine()

		switch line {
		case "yes", "y":
			s.MessageSink <- ""
			return nil
		case "all", "a":
			s.autoConfirm = true
			s.MessageSink <- ""
			return nil
		}
		prefix = "Please enter 'yes' to confirm."
	}
}

func (s *stateUI) Warningf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(s.bypassWriter, strings.TrimSuffix(msg, "\n"))
}

func (s *stateUI) Verbosef(format string, args ...interface{}) {
	if s.verbose {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintln(s.bypassWriter, strings.TrimSuffix(msg, "\n"))
	}
}

func (s *stateUI) processSinks() {
	defer s.stopWait.Done()
	for {
		select {
		case hdr := <-s.HeaderSink:
			s.processHeader(hdr)
		case evt := <-s.EventSink:
			s.processEvent(evt)
		case msg := <-s.MessageSink:
			s.processMessage(msg)
		case <-s.stopChan:
			return
		}
	}
}

func (s *stateUI) processHeader(header string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.lastHeader = header
	s.redraw()
}

func (s *stateUI) processEvent(evt scheduler.Event) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.states[evt.UnitName] = evt.Message
	s.redraw()
}

func (s *stateUI) processMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.lastMessage = message
	s.redraw()
}

func (s *stateUI) redraw() {
	msg := ""
	if s.lastHeader != "" {
		msg = msg + strings.TrimSuffix(s.lastHeader, "\n") + "\n\n"
	}
	if len(s.states) > 0 {
		lines := []string{"# Unit | State"}
		for unitName, state := range s.states {
			line := fmt.Sprintf("# %s | %s", unitName, state)
			lines = append(lines, line)
		}
		sort.Strings(lines)
		formattedLines := strings.Replace(columnize.SimpleFormat(lines), "#", " ", -1)
		msg = msg + "Status\n" + formattedLines + "\n\n"
	}
	if s.lastMessage != "" {
		msg = msg + strings.TrimSuffix(s.lastMessage, "\n") + "\n\n"
	}
	fmt.Fprintln(s.writer, strings.TrimSuffix(msg, "\n"))
	s.writer.Flush()
}
