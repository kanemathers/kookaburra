package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
)

func main() {
	httpPort := flag.Int("http", 8080, "Port to bind on for HTTP connections")
	workingDir := flag.String("dir", os.TempDir(), "Directory to store downloaded data")
	cleanup := flag.Bool("cleanup", true, "Remove downloaded data on quit")
	chooseLargest := flag.Bool("largest", false, "Automatically play the largest file in the torrent")
	playAllFiles := flag.Bool("all", false, "Play all audio/video files in the torrent")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [torrent]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.Arg(0) == "" {
		log.Fatal("no torrent provided")
	}

	client, err := NewClient(&ClientConfig{
		WorkingDir: path.Join(*workingDir, "kookaburra"),
		Cleanup:    *cleanup,
		HTTPPort:   *httpPort,
		Readahead:  5 * 1024 * 1024,
	})

	if err != nil {
		log.Fatalf("creating client: %s\n", err)
	}

	defer client.Close()

	fmt.Printf("Fetching torrent...\n\n")

	torrent, err := client.LoadTorrent(flag.Arg(0))

	if err != nil {
		log.Fatalf("fetching torrent: %s", err)
	}

	go client.Render(*playAllFiles)

	if !*playAllFiles {
		var file *SeekableFile

		if !*chooseLargest {
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

			file = torrent.Files()[choice]
		} else {
			file = torrent.LargestFile()
		}

		log.Fatal(client.ServeFile(file))
	} else {
		log.Fatal(client.ServePlaylist())
	}
}
