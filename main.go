package main

import (
	"fmt"
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

const WorkingDir = "/mnt/nfs/working_dir/"

func generateDirs(taskId string) {
	os.MkdirAll(WorkingDir+taskId+"/input", 0744)
	os.MkdirAll(WorkingDir+taskId+"/output", 0744)
	os.MkdirAll(WorkingDir+taskId+"/code", 0744)
}

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
	generateDirs(jobID)
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

	err := r.ParseMultipartForm(100000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if r.FormValue("task-id") == "" {
		http.Error(w, "Missing task-id", http.StatusBadRequest)
		return
	}
	if r.FormValue("dir") == "" {
		http.Error(w, "Missing dir", http.StatusBadRequest)
		return
	}
	generateDirs(r.FormValue("task-id"))
	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// open destination
			var outfile *os.File
			destination := fmt.Sprintf("%v/%v/%v/%v", WorkingDir, r.Form.Get("task-id"), r.Form.Get("dir"), hdr.Filename)
			if outfile, err = os.Create(destination); nil != err {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Copy
			var written int64
			if written, err = io.Copy(outfile, infile); nil != err {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Println("uploaded file:" + destination + ";length:" + strconv.Itoa(int(written)))
		}
	}
}

func main() {
	http.Handle("/run", websocket.Handler(handleRun))
	http.HandleFunc("/upload", handleUpload)
	http.Handle("/assets/", http.FileServer(http.Dir("./")))
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(WorkingDir))))
	http.Handle("/", http.FileServer(http.Dir("./templates")))
	log.Println("Starting server ..")
	log.Fatal(http.ListenAndServe(":5055", nil))
}
