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

	var wg sync.WaitGroup
	log := NewLogger("[Worker]")
	threads := flag.Int("t", 0, "number of threads")
	host := flag.String("c", "", "controller host")
	port := flag.Int("p", 0, "controller port")

	flag.Parse()
	if *host == "" || *port <= 0 || *threads <= 0 {
		flag.Usage()
		log.Fatal("Usage: worker -c HOST -p PORT -t THREADS")
	}

	address := fmt.Sprintf("localhost:%d", *port)
	conn, err := net.Dial(protocol.TCP, address)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()
	log.Println("connected to controller")

	writeCh := make(chan WriteMsg, 4)
	jobCh := make(chan *protocol.CrackingJob, 1)

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	wg.Add(2)

	var delta_tested int64
	var total_tested int64

	go func() {
		defer wg.Done()
		writeRequests(encoder, writeCh, log)
	}()

	go func() {
		defer wg.Done()
		readRequests(decoder, writeCh, jobCh, &delta_tested, &total_tested, log)
	}()

	writeCh <- protocol.WorkerMessage{Status: protocol.READY}
	log.Printf("-> sent %s", protocol.READY)

	job := <-jobCh
	log.Printf("job received: id=%d user=%s", job.Id, job.Username)
	jobReceiveEnd := time.Now()

	var totalCrackTime time.Duration
	crackStart := time.Now()
	var password string

	indices := []int{0}

	for {
		// build candidate
		buf := make([]rune, len(indices))
		for i, idx := range indices {
			buf[i] = charset[idx]
		}
		test := string(buf)

		// fmt.Printf("Next Password: %s\n", test)
		found, err := crackPassword(job, test)

		atomic.AddInt64(&total_tested, 1)
		atomic.AddInt64(&delta_tested, 1)
		if err != nil {

			encoder.Encode(protocol.WorkerMessage{Status: protocol.FAILED})
			log.Printf("→ Sent %s", protocol.FAILED)
			log.Printf("error occurred during cracking password")
			continue
		}

		if found {
			totalCrackTime = time.Since(crackStart)
			log.Printf("Password found %s\n", test)
			password = test
			break
		}

		indices = nextPassword(indices)
	}

	resultsSentStart := time.Now()
	result := protocol.CrackResult{
		Password: password,
		Metrics: protocol.WorkerMetrics{
			TotalCrackingTimeNanos: totalCrackTime.Nanoseconds(),
			WorkerReceiveJobNanos:  jobReceiveEnd.UnixNano(),
			WorkerSentResultsNanos: resultsSentStart.UnixNano(),
		},
	}

	log.Printf("result ready: password=%q crackTime=%v", result.Password, time.Duration(result.Metrics.TotalCrackingTimeNanos))

	log.Println("sending result")
	writeCh <- protocol.WorkerMessage{Status: protocol.SUCCESS}
	writeCh <- result
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
