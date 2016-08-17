package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	os.Setenv("PS1", "🐳  $ ")
	os.Setenv("DOCKER_HOST", "unix:///tmp/docker.sock")

	tunnelProcess := createTunnel()
	defer tunnelProcess.Kill()

	bash := exec.Command("bash")

	bash.Stdout = os.Stdout
	bash.Stderr = os.Stderr
	bash.Stdin = os.Stdin

	err := bash.Run()
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func createTunnel() *os.Process {

	user := ""
	host := ""

	os.RemoveAll("/tmp/docker.sock")

	if len(os.Args) == 2 {
		host = os.Args[1]
		user = "root"
	} else if len(os.Args) == 3 {
		user = os.Args[1]
		host = os.Args[2]
	} else {
		log.Fatalln("usage: docker-tunnel [user] host")
		return nil
	}

	cmd := exec.Command("ssh", "-nNT", "-L", "/tmp/docker.sock:/var/run/docker.sock", user+"@"+host)
	err := cmd.Start()
	if err != nil {
		log.Fatalln("ERROR:", err.Error())
	}

	return cmd.Process
}
