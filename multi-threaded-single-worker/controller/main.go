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

func humanDuration(d time.Duration) string {
	abs := d.Abs()

	switch {
	case abs < time.Microsecond:
		return fmt.Sprintf("%d ns", d.Nanoseconds())
	case abs < time.Second:
		return fmt.Sprintf("%.3f ms", float64(d.Microseconds())/1000)
	default:
		return fmt.Sprintf("%.3f s", d.Seconds())
	}
}

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

	// Parsing shadow file
	parseStart := time.Now()
	job, err := protocol.FindUserInShadow(*shadowFile, *username)
	if err != nil {
		log.Fatalf("failed to create job: %v", err)
	}
	job.Interval = *heartbeats
	parseTime := time.Since(parseStart)

	log.Printf("Job Created")
	log.Printf("\tId: %d", job.Id)
	log.Printf("\tInterval: %d", job.Interval)
	log.Printf("\tUsername: %s", job.Username)
	log.Printf("\tSettings: %s", job.Setting)
	log.Printf("\tFullHash: %s", job.FullHash)

	address := fmt.Sprintf(":%d", *port)
	ln, err := net.Listen("tcp", address)
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

	resultCh := make(chan ResultMsg)
	writeCh := make(chan protocol.Message)

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

	log.Println("sending shutdown")
	writeCh <- protocol.Message{Command: protocol.MsgShutdown}
	close(writeCh)

	endToEnd := time.Since(start)

	fmt.Println("\n==== Cracking Results ====")
	if result.Password != "" {
		fmt.Println("Password Found:", result.Password)
	} else {
		fmt.Println("Password Not Found")
	}

	fmt.Println("\n==== Metrics ====")
	fmt.Printf("Controller parse time:    %s\n", humanDuration(parseTime))
	fmt.Printf("Job dispatch latency:     %s\n", humanDuration(result.Metrics.JobDispatch))
	fmt.Printf("Worker cracking time:     %s\n", humanDuration(result.Metrics.WorkerCrack))
	fmt.Printf("Result return latency:    %s\n", humanDuration(result.Metrics.ResultReturn))
	fmt.Printf("End-to-end runtime:       %s\n", humanDuration(endToEnd))

	wg.Wait()
	conn.Close()
	os.Exit(0)
}
