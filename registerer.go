package main

import "os/exec"

type Job struct {
	id      string
	command string
}

var currentlyRunning map[Job]*exec.Cmd

func init() {
	currentlyRunning = make(map[Job]*exec.Cmd)
}

func RegisterJob(ID, command string, cmd *exec.Cmd) {
	currentlyRunning[Job{ID, command}] = cmd
}

func UnregisterJob(ID, command string) {
	delete(currentlyRunning, Job{ID, command})
}

func GetAllJobs(ID string) []string {
	ret := []string{}
	for k, _ := range currentlyRunning {
		if k.id == ID {
			ret = append(ret, k.command)
		}
	}
	return ret
}
