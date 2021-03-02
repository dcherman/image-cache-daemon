package main

import (
	"log"
	"os"

	"github.com/dcherman/image-cache-daemon/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdown(cmd.NewImageCacheDaemonCommand(), os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
