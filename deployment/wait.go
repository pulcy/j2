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

package deployment

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	sleepMutex  sync.Mutex
	sleepCond   *sync.Cond
	stopCounter int32
	waiting     bool
)

func init() {
	sleepCond = sync.NewCond(&sleepMutex)

	go func() {
		c := make(chan os.Signal, 10)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		for {
			select {
			case <-c:
				if !waiting {
					if stopCounter >= 1 {
						os.Exit(1)
					} else {
						fmt.Print("Press Ctrl-C again to stop\n")
						atomic.AddInt32(&stopCounter, 1)
					}
				}
				sleepCond.Broadcast()
			case <-time.After(time.Second * 5):
				atomic.StoreInt32(&stopCounter, 0)
			}
		}
	}()
}

// InterruptibleSleep holds execution for a given duration, or until an interrupt signal is received.
func InterruptibleSleep(duration time.Duration, message string) {
	waiting = true
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		delay := time.Millisecond * 500
		for {
			msg := fmt.Sprintf(message, time.Duration(time.Second*time.Duration(duration.Seconds())))
			fmt.Print(msg)
			if !waiting {
				fmt.Println()
				break
			}
			time.Sleep(delay)
			duration = duration - delay
			fmt.Print("\r")
			for _, _ = range msg {
				fmt.Print(" ")
			}
			fmt.Print("\r")
		}
	}()

	go func() {
		time.Sleep(duration)
		if waiting {
			sleepCond.Broadcast()
		}
	}()

	sleepMutex.Lock()
	sleepCond.Wait()
	sleepMutex.Unlock()

	atomic.StoreInt32(&stopCounter, 0)
	waiting = false

	wg.Wait()
}
