package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const downloadBuffer = 5 * 1024 * 1024

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

	dataDir := path.Join(*workingDir, "kookaburra")

	defer func() {
		if *cleanup {
			if err := os.RemoveAll(dataDir); err != nil {
				log.Printf("cleaning up directory: %s\n", err)
			}
		}
	}()

	client, err := NewClient(dataDir)

	if err != nil {
		log.Fatalf("creating client: %s\n", err)
	}

	defer client.Close()

	fmt.Printf("Fetching torrent...\n\n")

	torrent, err := client.LoadTorrent(flag.Arg(0))

	if err != nil {
		log.Fatalf("fetching torrent: %s", err)
	}

	router := mux.NewRouter()

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

		router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			http.ServeContent(writer, request, file.DisplayPath(), time.Now(), file)
		})
	} else {
		router.HandleFunc("/playlist.m3u", func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/vnd.apple.mpegurl")

			var buffer bytes.Buffer

			buffer.WriteString("#EXTM3U\n")

			for _, file := range torrent.Files() {
				mimetype := mime.TypeByExtension(filepath.Ext(file.DisplayPath()))

				// Crude method to detect filetypes. We don't want to be serving up NFO documents
				// or text files, images, etc...
				//
				// Would be nice to use http.DetectContentType but that requires we load 512 bytes
				// of the file before the detection can be performed.
				if strings.HasPrefix(mimetype, "video/") || strings.HasPrefix(mimetype, "audio/") {
					buffer.WriteString(fmt.Sprintf("#EXTINFO:0,%s\n", file.DisplayPath()))
					buffer.WriteString(fmt.Sprintf("http://127.0.0.1:%d/%s\n", *httpPort, file.DisplayPath()))
				}
			}

			if _, err := io.Copy(writer, &buffer); err != nil {
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

				return
			}
		})

		router.HandleFunc("/{filename}", func(writer http.ResponseWriter, request *http.Request) {
			for _, file := range torrent.Files() {
				if file.DisplayPath() == mux.Vars(request)["filename"] {
					http.ServeContent(writer, request, file.DisplayPath(), time.Now(), file)

					return
				}
			}

			http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		})
	}

	go client.Render(*httpPort, *playAllFiles)

	http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *httpPort), router)
}
