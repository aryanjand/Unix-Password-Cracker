package main

/*
#cgo LDFLAGS: -lcrypt
#include <stdlib.h>
#include <crypt.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"unsafe"

	"comp8005/internal/protocol"
	"comp8005/internal/utils"
)

var charset = []rune(
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789" +
		"@#%^&*()_+-=.,:;?",
)

func main() {

	log.SetPrefix("[Worker] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	args, err := utils.ParseWorkerFlags()
	log.Printf("Parsed worker args: host=%s port=%d", args.ControllerHost, args.ControllerPort)
	if err != nil {
		log.Fatal(err)
	}

	address := fmt.Sprintf("%s:%d", args.ControllerHost, args.ControllerPort)
	log.Printf("Connecting to controller at %s", address)
	conn, err := net.Dial(protocol.TCP, address)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	// defer conn.Close()
	log.Printf("Connected to controller")

	encoder := json.NewEncoder(conn)
	msg := *&protocol.WorkerMessage{
		Status: protocol.IDLE,
	}

	if err := encoder.Encode(msg); err != nil {
		log.Printf("❌ Failed to → Sent %s", protocol.IDLE)
		return
	}
	log.Printf("→ Sent %s", protocol.IDLE)

	msg = *&protocol.WorkerMessage{
		Status: protocol.READY,
	}
	if err := encoder.Encode(msg); err != nil {
		log.Printf("❌ Failed to → Sent %s", protocol.READY)
		return
	}
	log.Printf("→ Sent %s", protocol.READY)

	var job *protocol.CrackingJob
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&job); err != nil {
		msg = protocol.WorkerMessage{Status: protocol.FAILED}
		if err := encoder.Encode(msg); err != nil {
			log.Printf("❌ Failed to send FAILED: %v", err)
			return
		}
		log.Printf("→ Sent %s", protocol.FAILED)
		log.Printf("error occurred during receiving job")
	}
	log.Printf("← Receiving cracking job")
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
			msg = protocol.WorkerMessage{Status: protocol.FAILED}
			if err := encoder.Encode(msg); err != nil {
				log.Printf("❌ Failed to send FAILED: %v", err)
				return
			}
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

	log.Printf("→ Sending %s", protocol.SUCCESS)
	msg = protocol.WorkerMessage{Status: protocol.SUCCESS}
	if err := encoder.Encode(msg); err != nil {
		log.Printf("❌ Failed to send SUCCESS: %v", err)
		return
	}

	log.Printf("→ Sending result payload")
	if err := encoder.Encode(result); err != nil {
		log.Printf("❌ Failed to send result: %v", err)
		return
	}

	log.Printf("✓ Result sent successfully")

	log.Printf("Waiting for shutdown")
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Shutdown not received: %v", err)
		os.Exit(1)
	}
	if msg.Status == protocol.SHUTDOWN {
		log.Printf("Shutdown received")
		os.Exit(0)
	}
	os.Exit(1)

}

func crackPassword(job *protocol.CrackingJob, candidate string) (bool, error) {
	cPass := C.CString(candidate)
	cHash := C.CString(job.FullHash)
	defer C.free(unsafe.Pointer(cPass))
	defer C.free(unsafe.Pointer(cHash))

	var data unsafe.Pointer
	var size C.int

	res := C.crypt_ra(cPass, cHash, &data, &size)
	if res == nil {
		return false, fmt.Errorf("crypt_ra failed")
	}

	generated := C.GoString(res)
	return generated == job.FullHash, nil
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
