package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	humanize "github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type ClientConfig struct {
	WorkingDir string
	Cleanup    bool
	HTTPPort   int
	Readahead  int64
}

type Client struct {
	client  *torrent.Client
	torrent *Torrent

	config     *ClientConfig
	downloaded int64
	uploaded   int64
}

func NewClient(config *ClientConfig) (*Client, error) {
	torrentClient, err := torrent.NewClient(&torrent.Config{
		DataDir: config.WorkingDir,
		DHTConfig: dht.ServerConfig{
			StartingNodes: dht.GlobalBootstrapAddrs,
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "creating torrent client")
	}

	return &Client{
		client: torrentClient,
		config: config,
	}, nil
}

func (self *Client) Close() {
	self.torrent.Drop()
	self.client.Close()

	if self.config.Cleanup {
		if err := os.RemoveAll(self.config.WorkingDir); err != nil {
			log.Printf("cleaning up dir: %s: %v\n", self.config.WorkingDir, err)
		}
	}
}

func (self *Client) LoadTorrent(path string) (*Torrent, error) {
	var t *torrent.Torrent
	var err error

	if strings.HasPrefix(path, "magnet:") {
		t, err = self.client.AddMagnet(path)

		if err != nil {
			return nil, errors.Wrap(err, "adding magnet")
		}
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

		t, err = self.client.AddTorrent(metaInfo)

		if err != nil {
			return nil, errors.Wrap(err, "adding torrent")
		}
	} else {
		metaInfo, err := metainfo.LoadFromFile(path)

		if err != nil {
			return nil, errors.Wrap(err, "loading torrent file from path")
		}

		t, err = self.client.AddTorrent(metaInfo)

		if err != nil {
			return nil, errors.Wrap(err, "adding torrent")
		}
	}

	<-t.GotInfo()

	self.torrent = &Torrent{
		Torrent:   t,
		readahead: self.config.Readahead,
	}

	return self.torrent, nil
}

func (self *Client) PercentageComplete() float64 {
	info := self.torrent.Info()

	if info == nil {
		return 0
	}

	return float64(self.torrent.BytesCompleted()) / float64(info.TotalLength()) * 100
}

func (self *Client) Render(playAllFiles bool) {
	var clear string

	if runtime.GOOS == "windows" {
		clear = "cls"
	} else {
		clear = "clear"
	}

	for range time.Tick(1 * time.Second) {
		if self.torrent.Info() == nil {
			continue
		}

		downloaded := self.torrent.BytesCompleted()
		downloadSpeed := humanize.Bytes(uint64(downloaded - self.downloaded))
		self.downloaded = downloaded

		complete := humanize.Bytes(uint64(downloaded))
		size := humanize.Bytes(uint64(self.torrent.Info().TotalLength()))

		clearCmd := exec.Command(clear)
		clearCmd.Stdout = os.Stdout

		clearCmd.Run()

		fmt.Println(self.torrent.Name())
		fmt.Println(strings.Repeat("=", len(self.torrent.Name())))

		if downloaded > 0 {
			fmt.Printf("Progress: \t%s / %s  %.2f%%\n", complete, size, self.PercentageComplete())
		}

		if downloaded < self.torrent.Info().TotalLength() {
			fmt.Printf("Download speed: %s\n", downloadSpeed)
		}

		if !playAllFiles {
			if downloaded >= self.config.Readahead {
				fmt.Printf("\nOpen your media player and enter http://127.0.0.1:%d as the network address.\n", self.config.HTTPPort)
			} else {
				fmt.Printf("\nBuffering start of movie. Please wait...\n")
			}
		} else {
			fmt.Printf("\nLoad this M3U playlist into your media player http://127.0.0.1:%d/playlist.m3u\n", self.config.HTTPPort)
		}
	}
}

func (self *Client) ServeFile(file *SeekableFile) error {
	router := mux.NewRouter()

	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer, request, file.DisplayPath(), time.Now(), file)
	})

	return http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", self.config.HTTPPort), router)
}

func (self *Client) ServePlaylist() error {
	router := mux.NewRouter()

	router.HandleFunc("/playlist.m3u", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/vnd.apple.mpegurl")

		var buffer bytes.Buffer

		buffer.WriteString("#EXTM3U\n")

		for _, file := range self.torrent.Files() {
			mimetype := mime.TypeByExtension(filepath.Ext(file.DisplayPath()))

			// Crude method to detect filetypes. We don't want to be serving up NFO documents
			// or text files, images, etc...
			//
			// Would be nice to use http.DetectContentType but that requires we load 512 bytes
			// of the file before the detection can be performed.
			if strings.HasPrefix(mimetype, "video/") || strings.HasPrefix(mimetype, "audio/") {
				buffer.WriteString(fmt.Sprintf("#EXTINFO:0,%s\n", file.DisplayPath()))
				buffer.WriteString(fmt.Sprintf("http://127.0.0.1:%d/%s\n", self.config.HTTPPort, file.DisplayPath()))
			}
		}

		if _, err := io.Copy(writer, &buffer); err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	})

	router.HandleFunc("/{filename}", func(writer http.ResponseWriter, request *http.Request) {
		for _, file := range self.torrent.Files() {
			if file.DisplayPath() == mux.Vars(request)["filename"] {
				http.ServeContent(writer, request, file.DisplayPath(), time.Now(), file)

				return
			}
		}

		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})

	return http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", self.config.HTTPPort), router)
}
