package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/ukfast/gitlab-registry-cleanup/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("Failed to execute command: %s", err)
	}
}
