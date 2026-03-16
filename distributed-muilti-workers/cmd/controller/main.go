package main

import (
	"fmt"
	"net"
	"os"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/config"
	"github.com/aryanjand/Unix-Password-Cracker/internal/controller"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

const MAX_WORKERS = 10

func main() {
	log := utils.NewLogger("[Controller]")

	cfg, err := config.ParseController(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	shadow, err := controller.FindUserInShadow(cfg.ShadowFilePath, cfg.Username)
	if err != nil {
		log.Fatalf("failed to parse shadow file: %v", err)
	}

	log.Printf("shadow entry:\n"+"\tUsername: %s\n"+"\tsettings: %s\n"+"\tfullHash: %s",
		shadow.Username, shadow.Settings, shadow.FullHash,
	)

	address := fmt.Sprintf(":%d", cfg.Port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening for workers on %s", address)

	manager := controller.NewWorkerManger()
	alloc := chunk.NewChunkAllocator(uint64(cfg.PartitionSize), 0, 0)
	foundResultCh := make(chan string)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		remoteAddr := conn.RemoteAddr().String()
		workerLog := utils.NewLogger(fmt.Sprintf("[Controller][Worker: (%s)]", remoteAddr))

		worker := controller.NewWorker(conn, shadow, alloc, cfg.HeartbeatInterval, foundResultCh)
		manager.AddWorker(remoteAddr, *worker)
		workerLog.Printf("worker connected, total connected %d", manager.Count())

	}
}
