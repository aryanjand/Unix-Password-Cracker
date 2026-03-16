package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
	"github.com/aryanjand/Unix-Password-Cracker/internal/utils"
	"github.com/aryanjand/Unix-Password-Cracker/internal/worker"
)

const MAX_JOBS = 1

func main() {

	// Parse arguments
	log := utils.NewLogger("[Worker]")
	port := flag.Int("p", 0, "controller port")
	host := flag.String("c", "", "controller host")
	threads := flag.Int("t", 0, "number of threads")
	partition := flag.Int("s", 1, "partition size for password space")

	flag.Parse()
	if *host == "" || *port <= 0 || *partition <= 0 || *port > 65535 || *threads <= 0 {
		flag.Usage()
		log.Fatal("Usage: worker -c HOST -p PORT -t THREADS")
	}

	// Connect to the controller
	address := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := net.Dial("tcp", address)

	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()
	log.Println("connected to controller")

	w := worker.NewWorker(conn)
	log.Print("created worker")
	var result string

	for {

		// Todo left
		// 2. Stop the workers now
		// 3. Send Found Results, Metrics

		// 1. Sent a Job Request
		w.Conn.Send <- protocol.Message{Command: protocol.MsgJobReq}
		log.Printf("-> sent %s", protocol.MsgJobReq)

		// 2. Wait for the Job Response
		job := <-w.JobCh
		// jobLog.Printf("job started (index range=%d-%d, password range=(%s-%s), workers=%d)",
		// 	job.StartIndex, job.EndIndex, generateNextPassword(job.StartIndex), generateNextPassword(job.EndIndex), workers)

		// 3. Get a Job Runner
		runner := worker.NewJobRunner(job)

		// 4. Run using multi threading
		result = runner.Run(*threads)

		if result != "" {
			break
		}

	}

	fmt.Println("Result %s ", result)
	os.Exit(0)
}
