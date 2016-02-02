package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const WorkingDir = "/mnt/nfs/working_dir/"
const newLine = "\n====================================================================\n"

func generateDirs(taskId string) {
	os.MkdirAll(WorkingDir+taskId+"/input", 0755)
	os.MkdirAll(WorkingDir+taskId+"/output", 0755)
	os.MkdirAll(WorkingDir+taskId+"/code", 0755)
	os.MkdirAll(WorkingDir+taskId+"/stdfiles", 0755)
}

func runCommand(ws *websocket.Conn, jobID, command, cpus, memory string) {
	generateDirs(jobID)
	rClusterJobId := rand.Intn(100000) + 100000
	websocket.Message.Send(ws, "STDOUT: $ "+command+"\n")
	websocket.Message.Send(ws, "STDOUT: Your r cluster id "+fmt.Sprintf("%v", rClusterJobId)+"\n")
	websocket.Message.Send(ws, "STDERR: $ "+command+"\n")
	cmd := exec.Command("setsid", "./scripts/run-r-script.sh", "--name", jobID, "--command", command, "--cpus", cpus, "--memory", memory)
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
	if err != nil {
		log.Println(err.Error())
		websocket.Message.Send(ws, "STDERR: "+err.Error()+"\nSTDOUT: "+newLine)
		return
	}
	RegisterJob(jobID, command, rClusterJobId, cmd)

	prefixerFuction := func(prefix string, r io.Reader, w *websocket.Conn, lock sync.Mutex) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lock.Lock()
			err := websocket.Message.Send(w, prefix+scanner.Text()+"\n")
			lock.Unlock()
			if err != nil {
				return
			}
		}
	}

	pinger := func(ws *websocket.Conn, lock sync.Mutex) {
		for {
			lock.Lock()
			err := websocket.Message.Send(ws, "PING")
			lock.Unlock()
			if err != nil {
				return
			}
			time.Sleep(time.Second * 2)
		}
	}

	lock := *new(sync.Mutex)
	go prefixerFuction("STDOUT: ", stdout, ws, lock)
	go prefixerFuction("STDERR: ", stderr, ws, lock)
	go pinger(ws, lock)

	cmd.Wait()
	UnregisterJob(jobID, command, rClusterJobId)

	websocket.Message.Send(ws, "STDOUT: "+newLine)
	websocket.Message.Send(ws, "STDERR: "+newLine)
	log.Printf("Command '%v'@%v Done..", command, jobID)
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
	cpus := tmp[1]
	memory := tmp[2]
	command := strings.Join(tmp[3:], " ")

	runCommand(ws, jobID, command, cpus, memory)

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

func handleRunning(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	taskId := r.URL.Query().Get("task-id")
	if taskId == "" {
		http.Error(w, "Missing task-id", http.StatusBadRequest)
		return
	}
	commands := GetAllJobs(taskId)

	jobs := []string{}
	for _, v := range commands {

		jobs = append(jobs, fmt.Sprintf("%v", v))
	}
	w.Write([]byte(strings.Join(jobs, "<br/>")))
}

func handleKill(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("Content-Type", "text/plain")
	rClusterId := r.URL.Query().Get("r-cluster-id")
	if rClusterId == "" {
		http.Error(w, "Missing r-cluster-id", http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(rClusterId)
	if KillJob(id) {
		w.Write([]byte("KILLED!"))
	} else {
		w.Write([]byte("Id not found!"))
	}

}

func main() {
	http.Handle("/run", websocket.Handler(handleRun))
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/running", handleRunning)
	http.HandleFunc("/kill", handleKill)
	http.Handle("/assets/", http.FileServer(http.Dir("./")))
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(WorkingDir))))
	http.Handle("/", http.FileServer(http.Dir("./templates")))
	log.Println("Starting server ..")
	log.Fatal(http.ListenAndServe(":5055", nil))
}
