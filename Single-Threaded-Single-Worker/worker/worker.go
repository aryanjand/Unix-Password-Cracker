package main

/*
#cgo LDFLAGS: -lcrypt
#include <stdlib.h>
#include <crypt.h>
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"

	"comp8005/internal/protocol"
)

func main() {

	// log.SetPrefix("[Worker] ")
	// log.SetFlags(log.LstdFlags | log.Lshortfile)

	// args, err := utils.ParseWorkerFlags()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// address := fmt.Sprintf("%s:%d", args.ControllerHost, args.ControllerPort)
	// conn, err := net.Dial(protocol.TCP, address)
	// if err != nil {
	// 	log.Fatal("connect error:", err)
	// }
	// defer conn.Close()

	// fmt.Fprintln(conn, protocol.IDLE)
	// log.Printf("sent %s", protocol.IDLE)

	// fmt.Fprintln(conn, protocol.READY)
	// log.Printf("sent %s", protocol.READY)

	// var job *protocol.CrackingJob
	// decoder := json.NewDecoder(conn)
	// if err := decoder.Decode(&job); err != nil {
	// 	fmt.Fprintln(conn, protocol.FAILED)
	// 	log.Printf("error occurred during receiving job")
	// }
	// log.Println("received job:", job)

	job := &protocol.CrackingJob{
		Id:       1,
		Username: "aryan",
		Setting:  "$6$rounds=5000$salt",
		FullHash: "npWxgGSDyHqVDl380f15BGx4ivQ6LFf9YBLRnIcuZD2OVBEZsuLseXPt1HMyGk1aZftzH0Klzh5LQULwLdJuN.",
	}

	password, err := crackPassword(job, "A")
	if err != nil {
		// fmt.Fprintln(conn, protocol.FAILED)
		log.Printf("error occurred during cracking password")
	}

	fmt.Printf("Check the password: ", password)

	// fmt.Fprintln(conn, protocol.SUCCESS)
	// fmt.Fprintln(conn, password)

}

func crackPassword(job *protocol.CrackingJob, candidate string) (bool, error) {
	cPass := C.CString(candidate)
	cSet := C.CString(job.Setting)
	defer C.free(unsafe.Pointer(cPass))
	defer C.free(unsafe.Pointer(cSet))

	var data unsafe.Pointer
	var size C.int

	res := C.crypt_ra(cPass, cSet, &data, &size)
	if res == nil {
		return false, fmt.Errorf("crypt_ra failed")
	}

	generated := C.GoString(res)
	return generated == job.FullHash, nil
}
