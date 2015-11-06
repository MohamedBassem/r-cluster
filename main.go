package main

import (
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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
	s := strings.Split(command, " ")
	websocket.Message.Send(ws, "$ "+command+"\n")
	cmd := exec.Command(s[0], s[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer stderr.Close()

	err = cmd.Start()
	defer websocket.Message.Send(ws, "\n")
	if err != nil {
		websocket.Message.Send(ws, err.Error()+"\n")
		return
	}
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

func handleUpload(w http.ResponseWriter, r *http.Request) {

	const _24K = (1 << 20) * 24
	r.ParseMultipartForm(_24K)

	if r.FormValue("task-id") == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var err error
	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); nil != err {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// open destination
			var outfile *os.File
			if outfile, err = os.Create("/tmp/" + r.Form.Get("task-id") + "/output/" + hdr.Filename); nil != err {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// 32K buffer copy
			var written int64
			if written, err = io.Copy(outfile, infile); nil != err {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Println("uploaded file:" + hdr.Filename + ";length:" + strconv.Itoa(int(written)))
		}
	}
}

func main() {
	http.Handle("/run", websocket.Handler(handleRun))
	http.HandleFunc("/uploadcode", handleUpload)
	log.Println("Starting server ..")
	log.Fatal(http.ListenAndServe(":5055", nil))
}
