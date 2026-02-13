package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

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

func main() {
	start := time.Now()
	var wg sync.WaitGroup
	log := NewLogger("[Controller]")

	port := flag.Int("p", 0, "port to bind")
	username := flag.String("u", "", "username")
	shadowFile := flag.String("f", "", "shadow file path")
	heartbeats := flag.Int("b", 0, "heartbeat interval in seconds")

	flag.Parse()
	if *port <= 0 || *port > 65535 || *heartbeats <= 0 || *shadowFile == "" || *username == "" {
		flag.Usage()
		log.Fatal("Usage: controller -p PORT -f SHADOW_FILE -u USERNAME -b HEARTBEAT_SECONDS")
	}

	parseStart := time.Now()
	job, err := protocol.FindUserInShadow(*shadowFile, *username)
	if err != nil {
		log.Fatalf("failed to create job: %v", err)
	}
	job.Interval = *heartbeats
	parseTime := time.Since(parseStart)

	log.Printf("Job Created")
	log.Printf("	Id: %d", job.Id)
	log.Printf("	Interval: %d", job.Interval)
	log.Printf("	Username: %s", job.Username)
	log.Printf("	Settings: %s", job.Setting)

	address := fmt.Sprintf(":%d", *port)
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

	writeCh := make(chan WriteMsg)
	resultCh := make(chan ResultMsg)

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	wg.Add(2)
	go func() {
		defer wg.Done()
		writeRequests(encoder, *heartbeats, writeCh, log)
	}()

	go func() {
		defer wg.Done()
		readRequests(decoder, *job, writeCh, resultCh, log)
	}()

	result, ok := <-resultCh
	if !ok {
		log.Fatal("result channel closed")
	}

	if result.Err != nil {
		log.Fatal(result.Err)
	}

	endToEnd := time.Since(start)
	fmt.Println("\n==== Cracking Results ====")
	if result.Result.Password != "" {
		fmt.Println("Password Found:", result.Result.Password)
	} else {
		fmt.Println("Password Not Found")
	}

	fmt.Println("\n==== Metrics ====")
	fmt.Printf("Controller parse time:    %v\n", parseTime)
	fmt.Printf("Job dispatch latency:     %v\n", result.Metrics.JobDispatch)
	fmt.Printf("Worker cracking time:     %v\n", time.Duration(result.Result.Metrics.TotalCrackingTimeNanos))
	fmt.Printf("Result return latency:    %v\n", result.Metrics.ResultReturn)
	fmt.Printf("End-to-end runtime:       %v\n", endToEnd)

	log.Println("sending shutdown")
	writeCh <- protocol.WorkerMessage{Status: protocol.SHUTDOWN}
	close(writeCh)

	wg.Wait()
	conn.Close()
	os.Exit(0)
}
