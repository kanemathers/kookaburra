package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

func main() {
	httpAddr := flag.String("http", ":8080", "Address to bind on for HTTP connections")
	workingDir := flag.String("dir", os.TempDir(), "Directory to store downloaded data")
	cleanup := flag.Bool("cleanup", true, "Remove downloaded data on quit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [torrent]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.Arg(0) == "" {
		log.Fatal("no torrent provided")
	}

	_, httpPort, err := net.SplitHostPort(*httpAddr)

	if err != nil {
		log.Fatalf("invalid http address: %s\n", *httpAddr)
	}

	dataDir := path.Join(*workingDir, "kookaburra")

	defer func() {
		if *cleanup {
			if err := os.RemoveAll(dataDir); err != nil {
				log.Printf("cleaning up directory: %s\n", err)
			}
		}
	}()

	client, err := NewClient(*workingDir)

	if err != nil {
		log.Fatalf("creating client: %s\n", err)
	}

	defer client.Close()

	fmt.Println("Fetching torrent...")

	torrent, err := client.LoadTorrent(flag.Arg(0))

	if err != nil {
		log.Fatalf("fetching torrent: %s", err)
	}

	fmt.Println("Found these files in the torrent. Select which one you'd like to stream:")
	fmt.Println()

	for i, file := range torrent.Files() {
		fmt.Printf("    [%d] %s\n", i, file.DisplayPath())
	}

	fmt.Println()

	var choice int

	for {
		fmt.Printf("File: ")

		if _, err := fmt.Scanln(&choice); err != nil || choice < 0 || choice > len(torrent.Files()) {
			fmt.Println("Invalid choice")
		} else {
			break
		}
	}

	file := torrent.Files()[choice]

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer, request, file.DisplayPath(), time.Now(), file)
	})

	fmt.Printf("\nOpen your media player and enter http://127.0.0.1:%s as the network address.\n", httpPort)

	http.ListenAndServe(*httpAddr, nil)
}
