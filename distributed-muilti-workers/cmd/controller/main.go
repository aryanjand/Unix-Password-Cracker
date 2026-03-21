package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/chunk"
	"github.com/aryanjand/Unix-Password-Cracker/internal/config"
	"github.com/aryanjand/Unix-Password-Cracker/internal/controller"
	"github.com/aryanjand/Unix-Password-Cracker/internal/persistence"
	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/storage"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
)

func main() {
	log := utils.NewLogger("[Controller]")

	cfg, err := config.ParseController(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	stateStore, err := storage.NewMySQLStore(cfg.MySQLDSN, cfg.Reset)
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}
	defer func() {
		if err := stateStore.Close(); err != nil {
			log.Printf("failed to close mysql store: %v", err)
		}
	}()
	log.Printf("connected to mysql")

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

	alloc := chunk.NewChunkAllocator(uint64(cfg.PartitionSize), 0, 0)
	listenerCtx, listenerCancel := context.WithCancel(context.Background())

	manager := controller.NewWorkerManger()
	foundResultCh := make(chan string)
	connCh := make(chan net.Conn)
	var password string

	go func() {
		defer close(connCh)
		for {
			conn, err := ln.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Printf("accept error: %v", err)
				continue
			}

			select {
			case connCh <- conn:
			case <-listenerCtx.Done():
				_ = conn.Close()
				return
			}
		}
	}()

	for password == "" {
		select {
		case password = <-foundResultCh:
			listenerCancel()
			_ = ln.Close()
		case conn, ok := <-connCh:
			if !ok {
				break
			}

			remoteAddr := conn.RemoteAddr().String()
			workerLog := utils.NewLogger(fmt.Sprintf("[Controller][Worker: (%s)]", remoteAddr))

			worker := controller.NewWorker(
				conn,
				remoteAddr,
				shadow,
				alloc,
				cfg.HeartbeatInterval,
				cfg.Checkpoint,
				foundResultCh,
				stateStore,
				workerLog,
			)
			manager.AddWorker(remoteAddr, *worker)
			workerLog.Printf("worker connected, total connected %d", manager.Count())
		}
	}

	log.Println("Found Result ", password)
	_ = stateStore.UpsertWorkerState(context.Background(), "controller", persistence.WorkerStateCompleted, "")
	manager.BroadcastMessage(protocol.MsgStop)

	time.Sleep(5 * time.Second)

}
