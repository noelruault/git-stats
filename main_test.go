package main

import (
	"log"
	"os/exec"
	"strings"
	"testing"
)

func TestHtmlToImage(t *testing.T) {
	command := "npx node-html-to-image-cli ./out/charts/lines.html ./out/images/lines.png"
	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		panic(err)
	}

	log.Print(string(data))
}
