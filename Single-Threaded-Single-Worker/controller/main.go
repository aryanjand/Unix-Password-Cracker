package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
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
	log := NewLogger("[Controller]")
	port := flag.Int("p", 0, "port to bind")
	username := flag.String("u", "", "username")
	shadowFile := flag.String("f", "", "shadow file path")
	flag.Parse()
	if *port <= 0 || *shadowFile == "" || *username == "" {
		log.Fatal("Usage: controller -p PORT -f SHADOW_FILE -u USERNAME")
	}

	go func() {
		log.Println("pprof listening on :3000")
		http.ListenAndServe("localhost:3000", nil)
	}()

	parseStart := time.Now()
	job, err := protocol.FindUserInShadow(*shadowFile, *username)
	if err != nil {
		log.Fatalf("failed to create job: %v", err)
	}
	parseTime := time.Since(parseStart)

	log.Printf("Job Created")
	log.Printf("	Id: %d", job.Id)
	log.Printf("	Username: %s", job.Username)
	log.Printf("	Settings: %s", job.Setting)

	address := fmt.Sprintf("localhost:%d", *port)
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

	encoder := json.NewEncoder(conn)
	result, metrics, err := handleWorkerConnection(conn, job, log)
	if err != nil {
		log.Fatal(err)
	}

	endToEnd := time.Since(start)
	fmt.Println("\n==== Cracking Results ====")
	if result.Password != "" {
		fmt.Println("Password Found:", result.Password)
	} else {
		fmt.Println("Password Not Found")
	}

	fmt.Println("\n==== Metrics ====")
	fmt.Printf("Controller parse time:    %v\n", parseTime)
	fmt.Printf("Job dispatch latency:     %v\n", metrics.JobDispatch)
	fmt.Printf("Worker cracking time:     %v\n", time.Duration(result.Metrics.TotalCrackingTimeNanos))
	fmt.Printf("Result return latency:    %v\n", metrics.ResultReturn)
	fmt.Printf("End-to-end runtime:       %v\n", endToEnd)

	encoder.Encode(protocol.WorkerMessage{Status: protocol.SHUTDOWN})
	log.Printf("Shutdown Sent")

	conn.Close()
	os.Exit(0)
}
