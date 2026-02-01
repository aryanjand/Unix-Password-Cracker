package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"comp8005/internal/protocol"
	"comp8005/internal/utils"
)

func main() {
	processStarted := time.Now()
	parseStart := time.Now()
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

	parseTime := time.Since(parseStart)

	log.Printf("Job Created")
	log.Printf("	Id: %d", job.Id)
	log.Printf("	Username: %s", job.Username)
	log.Printf("	Settings: %s", job.Setting)
	log.Printf("	FullHash: %s", job.FullHash)

	address := fmt.Sprintf(":%d", args.Port)
	ln, err := net.Listen(protocol.TCP, address)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Printf("Listening for workers on %s", address)

	conn, err := ln.Accept()
	if err != nil {
		log.Println("accept error:", err)
	}
	log.Printf("Worker connected from %s", conn.RemoteAddr())
	defer conn.Close()

	result, dispatchLatency, returnLatency, err := handleWorkerConnection(conn, job)
	if err != nil {
		log.Fatal(err)
	}

	if result.Password != "" {
		log.Println("Password Found: ", result.Password)
	} else {
		log.Println("Password Not Found")
	}

	endToEnd := time.Since(processStarted)
	fmt.Println("\n==== Cracking Results ====")
	if result.Password != "" {
		fmt.Println("Password Found:", result.Password)
	} else {
		fmt.Println("Password Not Found")
	}

	fmt.Println("\n==== Metrics ====")
	fmt.Printf("Controller parse time:    %v\n", parseTime)
	fmt.Printf("Job dispatch latency:     %v\n", dispatchLatency)
	fmt.Printf("Worker cracking time:     %v\n",
		time.Duration(result.Metrics.TotalCrackingTimeNanos))
	fmt.Printf("Result return latency:    %v\n", returnLatency)
	fmt.Printf("End-to-end runtime:       %v\n", endToEnd)

	encoder := json.NewEncoder(conn)
	msg := protocol.WorkerMessage{Status: protocol.SHUTDOWN}
	log.Printf("→ Sending Shutdown")
	if err := encoder.Encode(msg); err != nil {
		log.Printf("❌ Failed to send SHUTDOWN: %v", err)
		return
	}
	log.Printf("Shutdown Sent")
	os.Exit(0)

}

func handleWorkerConnection(conn net.Conn, job *protocol.CrackingJob) (*protocol.CrackResult, time.Duration, time.Duration, error) {

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	var (
		result            *protocol.CrackResult
		jobSentStart      time.Time
		resultsReceiveEnd time.Time
	)
	for {
		var msg protocol.WorkerMessage
		if err := decoder.Decode(&msg); err != nil {
			log.Fatal(err)
		}

		cmd := msg.Status
		log.Printf("← Received %s", cmd)

		switch cmd {

		case protocol.IDLE:
			conn.SetReadDeadline(time.Time{})

		case protocol.READY:
			log.Printf("→ Sending cracking job to worker")
			jobSentStart = time.Now()
			if err := encoder.Encode(job); err != nil {
				return nil, 0, 0, err
			}

		case protocol.SUCCESS:
			log.Printf("← Receiving cracking result")
			if err := decoder.Decode(&result); err != nil {
				return nil, 0, 0, err
			}
			log.Printf("✓ Result received (password=%q)", result.Password)

			resultsReceiveEnd = time.Now()
			jobDispatchLatency := time.Unix(0, result.Metrics.WorkerReceiveJobNanos).Sub(jobSentStart)
			resultReturnLatency := resultsReceiveEnd.Sub(time.Unix(0, result.Metrics.WorkerSentResultsNanos))
			return result, jobDispatchLatency, resultReturnLatency, nil

		case protocol.FAILED:
			log.Printf("❌ Worker reported FAILURE")
			return nil, 0, 0, fmt.Errorf("worker reported failure")

		default:
			log.Printf("⚠ Unknown command from worker: %q", cmd)
		}
	}

}
