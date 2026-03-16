package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/controller"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

const MAX_WORKERS = 10

func main() {

	log := utils.NewLogger("[Controller]")

	var partition int

	port := flag.Int("p", 0, "port to bind")
	username := flag.String("u", "", "username")
	shadowFile := flag.String("f", "", "shadow file path")
	heartbeats := flag.Int("b", 0, "heartbeat interval in seconds")
	flag.IntVar(&partition, "c", 1, "partition size for password space")
	flag.IntVar(&partition, "s", 1, "partition size for password space")

	flag.Parse()
	if *port <= 0 || *port > 65535 || partition <= 0 || *heartbeats <= 0 || *shadowFile == "" || *username == "" {
		flag.Usage()
		log.Fatal("Usage: controller -p PORT -f SHADOW_FILE -u USERNAME -b HEARTBEAT_SECONDS -c PARTITION_SIZE")
	}

	shadow, err := controller.FindUserInShadow(*shadowFile, *username)
	if err != nil {
		log.Fatalf("failed to parse shadow file: %v", err)
	}

	log.Printf("shadow entry:\n"+"\tUsername: %s\n"+"\tsettings: %s\n"+"\tfullHash: %s",
		shadow.Username, shadow.Settings, shadow.FullHash,
	)

	address := fmt.Sprintf(":%d", *port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening for workers on %s", address)

	manager := controller.NewWorkerManger()
	alloc := chunk.NewChunkAllocator(uint64(partition), 0, 0)
	foundResultCh := make(chan string)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		remoteAddr := conn.RemoteAddr().String()
		workerLog := utils.NewLogger(fmt.Sprintf("[Controller][Worker: (%s)]", remoteAddr))

		worker := controller.NewWorker(conn, shadow, alloc, *heartbeats, foundResultCh)
		manager.AddWorker(remoteAddr, *worker)
		workerLog.Printf("worker connected, total connected %d", manager.Count())

	}

	os.Exit(0)
}
