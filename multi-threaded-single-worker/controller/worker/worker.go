package main

/*
#cgo LDFLAGS: -lcrypt
#include <stdlib.h>
#include <crypt.h>
*/
import "C"

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"controller/protocol"
)

type Logger struct {
	*log.Logger
}

func NewLogger(prefix string) *Logger {
	return &Logger{
		log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile),
	}
}

var charset = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "abcdefghijklmnopqrstuvwxyz" + "0123456789" + "@#%^&*()_+-=.,:;?")

func main() {

	// Parse arguments
	log := NewLogger("[Worker]")
	threads := flag.Int("t", 0, "number of threads")
	host := flag.String("c", "", "controller host")
	port := flag.Int("p", 0, "controller port")

	flag.Parse()
	if *host == "" || *port <= 0 || *port > 65535 || *threads <= 0 {
		flag.Usage()
		log.Fatal("Usage: worker -c HOST -p PORT -t THREADS")
	}

	// Connect to the controller
	address := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()
	log.Println("connected to controller")

	// Add Go routines to read and write to sockets (Handling Heartbeat request)
	var wg sync.WaitGroup
	writeCh := make(chan protocol.Message, 4)
	jobCh := make(chan *protocol.CrackingJob, 1)

	var delta_tested int64
	var total_tested int64

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	wg.Add(2)

	go func() {
		defer wg.Done()
		writeRequests(encoder, writeCh, log)
	}()

	go func() {
		defer wg.Done()
		readRequests(decoder, writeCh, jobCh, &delta_tested, &total_tested, log)
	}()

	writeCh <- protocol.Message{Command: protocol.MsgReady}
	log.Printf("-> sent %s", protocol.MsgReady)

	job := <-jobCh
	jobReceiveEnd := time.Now()

	log.Println("job received:")
	log.Printf("\tid: %d", job.Id)
	log.Printf("\tinterval: %d", job.Interval)
	log.Printf("\tusername: %s", job.Username)
	log.Printf("\tsettings: %s", job.Setting)
	log.Printf("\tfull hash: %s", job.FullHash)

	// Crack passwords
	done := make(chan struct{})
	resultCh := make(chan ResultMsg, 1)
	jobs := make(chan string, *threads)

	var totalCrackTime time.Duration
	crackStart := time.Now()

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				case test, ok := <-jobs:
					if !ok {
						return
					}

					found, err := crackPassword(job, test)
					atomic.AddInt64(&delta_tested, 1)
					atomic.AddInt64(&total_tested, 1)

					if err != nil {
						resultCh <- ResultMsg{Err: err}
						return
					}

					if found {
						resultCh <- ResultMsg{Found: test}
						close(done)
						return
					}
				}
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		indices := []int{0}

		for {
			select {
			case <-done:
				close(jobs)
				return
			default:
				buf := make([]rune, len(indices))
				for i, idx := range indices {
					buf[i] = charset[idx]
				}
				jobs <- string(buf)
				indices = nextPassword(indices)
			}
		}
	}()

	res := <-resultCh
	totalCrackTime = time.Since(crackStart)
	if res.Err != nil {
		log.Fatal("crack failed:", res.Err)
	}

	log.Printf("password found: %s", res.Found)
	resultsSentStart := time.Now()
	result := protocol.CrackResult{
		Password: res.Found,
		Metrics: protocol.WorkerMetrics{
			TotalCrackingTimeNanos: totalCrackTime.Nanoseconds(),
			WorkerReceiveJobNanos:  jobReceiveEnd,
			WorkerSentResultsNanos: resultsSentStart,
		},
	}

	log.Printf("result ready: password=%q crackTime=%v", result.Password, time.Duration(result.Metrics.TotalCrackingTimeNanos))
	log.Println("sending result")
	writeCh <- protocol.Message{
		Command: protocol.MsgResult,
		Result:  &result,
	}
	close(writeCh)

	wg.Wait()
	os.Exit(0)
}

func crackPassword(job *protocol.CrackingJob, candidate string) (bool, error) {
	data := C.struct_crypt_data{}
	cPass := C.CString(candidate)
	cHash := C.CString(job.FullHash)
	defer C.free(unsafe.Pointer(cPass))
	defer C.free(unsafe.Pointer(cHash))

	res := C.crypt_r(cPass, cHash, &data)
	if res == nil {
		return false, fmt.Errorf("crypt_r failed")
	}
	return C.GoString(res) == job.FullHash, nil
}

func nextPassword(p []int) []int {
	pos := len(p) - 1

	for pos >= 0 {
		p[pos]++
		if p[pos] < len(charset) {
			return p
		}
		p[pos] = 0
		pos--
	}

	return make([]int, len(p)+1)
}
