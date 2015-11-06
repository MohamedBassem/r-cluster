package main

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

func pinger(ws *websocket.Conn) {
	for {
		err := websocket.Message.Send(ws, "")
		if err != nil {
			return
		}
		time.Sleep(time.Second * 2)
	}
}

func runCommand(ws *websocket.Conn, jobID, command string) {
	cmd := exec.Command("ls", "-l")
	stdout, err := cmd.StdoutPipe()
	defer stdout.Close()
	if err != nil {
		log.Println(err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	defer stderr.Close()
	if err != nil {
		log.Println(err.Error())
		return
	}

	cmd.Start()
	defer cmd.Wait()

	go io.Copy(ws, stdout)
	go io.Copy(ws, stderr)
	go pinger(ws)
}

func handleRun(ws *websocket.Conn) {
	var message string
	log.Printf("Connection started\n")
	err := websocket.Message.Receive(ws, &message)
	if err != nil {
		log.Println("Connection Closed")
		return
	}
	log.Printf("Got: %v\n", message)
	tmp := strings.Split(message, " ")
	if len(tmp) < 2 {
		log.Printf("Recieved %v which is not in the format '<JobId> <command ...>'\n", message)
	}

	jobID := tmp[0]
	command := strings.Join(tmp[1:], " ")

	runCommand(ws, jobID, command)

}

func main() {
	http.Handle("/run", websocket.Handler(handleRun))
	log.Println("Starting server ..")
	log.Fatal(http.ListenAndServe(":5055", nil))
}
