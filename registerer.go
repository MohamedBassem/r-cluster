package main

import (
	"os/exec"
	"syscall"
)

type Job struct {
	id      int
	taskId  string
	command string
}

var currentlyRunning map[Job]*exec.Cmd

func init() {
	currentlyRunning = make(map[Job]*exec.Cmd)
}

func RegisterJob(ID, command string, rClusterId int, cmd *exec.Cmd) {
	currentlyRunning[Job{rClusterId, ID, command}] = cmd
}

func UnregisterJob(ID, command string, rClusterId int) {
	delete(currentlyRunning, Job{rClusterId, ID, command})
}

func GetAllJobs(taskId string) []Job {
	ret := []Job{}
	for k, _ := range currentlyRunning {
		if k.taskId == taskId {
			ret = append(ret, k)
		}
	}
	return ret
}

func KillJob(rClusterId int) bool {
	for k, v := range currentlyRunning {
		if k.id == rClusterId {
			syscall.Kill(-v.Process.Pid, syscall.SIGKILL)
			return true
		}
	}
	return false
}
