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

	log := NewLogger("[Worker]")

	host := flag.String("c", "", "controller host")
	port := flag.Int("p", 0, "controller port")
	flag.Parse()
	if *host == "" || *port <= 0 {
		log.Fatal("Usage: worker -c HOST -p PORT")
	}

	address := fmt.Sprintf("127.0.0.1:%d", *port)
	conn, err := net.Dial(protocol.TCP, address)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()
	log.Println("Connected to controller")

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	encoder.Encode(protocol.WorkerMessage{Status: protocol.READY})
	log.Printf("→ Sent %s", protocol.READY)

	var job *protocol.CrackingJob
	if err := decoder.Decode(&job); err != nil {
		encoder.Encode(protocol.WorkerMessage{Status: protocol.FAILED})
		log.Printf("→ Sent %s", protocol.FAILED)
		log.Fatalf("failed to receive job: %v", err)
	}
	log.Printf("Receiving job for user %s", job.Username)
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

		fmt.Printf("Next Password: %s\n", test)
		found, err := crackPassword(job, test)
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

	log.Printf("Result ready: password=%q crackTime=%v",
		result.Password,
		time.Duration(result.Metrics.TotalCrackingTimeNanos),
	)

	encoder.Encode(protocol.WorkerMessage{Status: protocol.SUCCESS})
	encoder.Encode(result)
	log.Println("Result sent")

	// Wait for SHUTDOWN
	var msg protocol.WorkerMessage
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err := decoder.Decode(&msg); err != nil {
		log.Println("Shutdown not received")
		os.Exit(1)
	}
	if msg.Status == protocol.SHUTDOWN {
		log.Println("Shutdown received, exiting")
	}

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
