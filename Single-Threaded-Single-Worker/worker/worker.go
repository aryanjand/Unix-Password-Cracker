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

	if err != nil {
		log.Fatal(err)
	}
	address := fmt.Sprintf("%s:%d", args.ControllerHost, args.ControllerPort)
	conn, err := net.Dial(protocol.TCP, address)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()

	fmt.Fprintln(conn, protocol.IDLE)
	log.Printf("sent %s", protocol.IDLE)

	fmt.Fprintln(conn, protocol.READY)
	log.Printf("sent %s", protocol.READY)

	var job *protocol.CrackingJob
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&job); err != nil {
		fmt.Fprintln(conn, protocol.FAILED)
		log.Printf("error occurred during receiving job")
	}

	var password string

	indices := []int{0} // starts at "A"

	for {
		// build candidate
		buf := make([]rune, len(indices))
		for i, idx := range indices {
			buf[i] = charset[idx]
		}
		test := string(buf)

		fmt.Println("Next Password: %s", test)
		found, err := crackPassword(job, test)
		if err != nil {
			fmt.Fprintln(conn, protocol.FAILED)
			log.Printf("error occurred during cracking password")
			continue
		}

		if found {
			password = test
			break
		}

		indices = nextPassword(indices)
	}

	fmt.Printf("Password Found %s\n", password)
	fmt.Fprintln(conn, protocol.SUCCESS)
	fmt.Fprintln(conn, password)

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

	// overflow → increase length (AA, AAA, etc.)
	return make([]int, len(p)+1)
}
