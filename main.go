package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/pkg/errors"
)

func fetchTorrent(client *torrent.Client, path string) (*torrent.Torrent, error) {
	if strings.HasPrefix(path, "magnet:") {
		t, err := client.AddMagnet(path)

		if err != nil {
			return nil, errors.Wrap(err, "adding torrent")
		}

		return t, nil
	} else if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		response, err := http.Get(path)

		if err != nil {
			return nil, errors.Wrap(err, "downloading torrent file")
		}

		defer response.Body.Close()

		metaInfo, err := metainfo.Load(response.Body)

		if err != nil {
			return nil, errors.Wrap(err, "loading metadata from torrent")
		}

		t, err := client.AddTorrent(metaInfo)

		if err != nil {
			return nil, errors.Wrap(err, "adding torrent")
		}

		return t, nil
	} else {
		metaInfo, err := metainfo.LoadFromFile(path)

		if err != nil {
			return nil, errors.Wrap(err, "loading torrent file from path")
		}

		t, err := client.AddTorrent(metaInfo)

		if err != nil {
			return nil, errors.Wrap(err, "adding torrent")
		}

		return t, nil
	}
}

type seekableTorrent struct {
	*torrent.Reader

	offset int64
	length int64
}

func (self *seekableTorrent) Seek(off int64, whence int) (ret int64, err error) {
	var pos int64

	switch whence {
	case io.SeekStart:
		pos = self.offset + off
	case io.SeekCurrent:
		pos = off
	case io.SeekEnd:
		pos = (self.offset + self.length) - off
	}

	return self.Reader.Seek(pos, whence)
}

func main() {
	httpPort := flag.String("http", ":8080", "Address to bind on for HTTP connections")
	dataDir := flag.String("data-dir", os.TempDir(), "Directory to store downloaded torrent data")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [torrent]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.Arg(0) == "" {
		log.Fatal("no torrent provided")
	}

	client, err := torrent.NewClient(&torrent.Config{
		DataDir: *dataDir,
		DHTConfig: dht.ServerConfig{
			StartingNodes: dht.GlobalBootstrapAddrs,
		},
	})

	if err != nil {
		log.Fatalf("creating client: %s", err)
	}

	defer client.Close()

	fmt.Println("Fetching torrent...")

	t, err := fetchTorrent(client, flag.Arg(0))

	if err != nil {
		log.Fatalf("fetching torrent: %s", err)
	}

	<-t.GotInfo()

	fmt.Println("Found these files in the torrent. Select which one you'd like to stream:")
	fmt.Println()

	for i, file := range t.Files() {
		fmt.Printf("    [%d] %s\n", i, file.DisplayPath())
	}

	fmt.Println()

	var choice int

	for {
		fmt.Printf("File: ")

		if _, err := fmt.Scanln(&choice); err != nil || choice < 0 || choice > len(t.Files()) {
			fmt.Println("Invalid choice")
		} else {
			break
		}
	}

	st := &seekableTorrent{
		Reader: t.NewReader(),
		offset: t.Files()[choice].Offset(),
		length: t.Files()[choice].Length(),
	}

	st.Reader.SetResponsive()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer, request, t.Files()[choice].DisplayPath(), time.Now(), st)
	})

	fmt.Printf("\nOpen your media player and enter http://127.0.0.1:8080 as the network address.\n")

	http.ListenAndServe(*httpPort, nil)
}
