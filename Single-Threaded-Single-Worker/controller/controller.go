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

	log.Println("Controller ready, waiting for workers on ", address)

	conn, err := ln.Accept()
	if err != nil {
		log.Println("accept error:", err)
	}
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
	os.Exit(0)

}

func handleWorkerConnection(conn net.Conn, job *protocol.CrackingJob) (*protocol.CrackResult, time.Duration, time.Duration, error) {

	reader := bufio.NewReader(conn)
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	var (
		result            *protocol.CrackResult
		jobSentStart      time.Time
		resultsReceiveEnd time.Time
	)
	for {

		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, 0, 0, err
		}

		cmd := strings.TrimSpace(line)

		switch cmd {

		case protocol.IDLE:
			conn.SetReadDeadline(time.Time{})
			continue

		case protocol.READY:
			log.Println(("Sending cracking job"))
			jobSentStart = time.Now()
			if err := encoder.Encode(job); err != nil {
				return nil, 0, 0, err
			}

		case protocol.SUCCESS:
			if err := decoder.Decode(&result); err != nil {
				return nil, 0, 0, err
			}
			resultsReceiveEnd = time.Now()
			jobDispatchLatency := time.Unix(0, result.Metrics.WorkerReceiveJobNanos).Sub(jobSentStart)
			resultReturnLatency := resultsReceiveEnd.Sub(time.Unix(0, result.Metrics.WorkerSentResultsNanos))
			return result, jobDispatchLatency, resultReturnLatency, nil

		case protocol.FAILED:
			return nil, 0, 0, fmt.Errorf("worker reported failure")

		default:
			log.Println("unknown command:", cmd)
		}

	}

}
