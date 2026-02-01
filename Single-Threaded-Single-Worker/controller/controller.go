package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"comp8005/internal/protocol"
	"comp8005/internal/utils"
)

func main() {
	log.SetPrefix("[Controller] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	args, err := utils.ParseControllerFlags()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Arguments Parsed")
	log.Printf("	Port: %d", args.Port)
	log.Printf("	Username: %s", args.Username)
	log.Printf("	Shadow file: %s", args.ShadowFile)

	job, err := protocol.FindUserInShadow(args.ShadowFile, args.Username)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Job Created")
	log.Printf("	Id: %d", job.Id)
	log.Printf("	Username: %s", job.Username)
	log.Printf("	Settings: %s", job.Setting)
	log.Printf("	FullHash: %s", job.FullHash)

	address := fmt.Sprintf("localhost:%d", args.Port)
	ln, err := net.Listen(protocol.TCP, address)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Println("Controller ready, waiting for workers on ", address)

	conn, err := ln.Accept()
	if err != nil {
		log.Println("accept error:", err)
	}
	defer conn.Close()

	password, err := handleWorkerConnection(conn, job)
	if err != nil {
		log.Fatal(err)
	}

	if password != "" {
		log.Println("Password Found: ", password)
	} else {
		log.Println("Password Not Found")
	}

	os.Exit(0)

}

func handleWorkerConnection(conn net.Conn, job *protocol.CrackingJob) (string, error) {

	reader := bufio.NewReader(conn)
	encoder := json.NewEncoder(conn)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	for {

		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("read error: ", err)
		}

		cmd := strings.TrimSpace(line)

		switch cmd {

		case protocol.IDLE:
			conn.SetReadDeadline(time.Time{})
			continue

		case protocol.READY:
			log.Println(("Sending cracking job"))
			if err := encoder.Encode(job); err != nil {
				return "", err
			}

		case protocol.SUCCESS:
			passwordLine, err := reader.ReadString('\n')
			if err != nil {
				log.Println("failed to read password: ", err)
			}
			return strings.TrimSpace(passwordLine), nil

		case protocol.FAILED:
			return "", fmt.Errorf("worker reported failure")

		default:
			log.Println("unknown command:", cmd)
		}

	}

}
