package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

type Client struct {
	client  *torrent.Client
	torrent *Torrent

	downloaded int64
	uploaded   int64
}

func NewClient(dataDir string) (*Client, error) {
	torrentClient, err := torrent.NewClient(&torrent.Config{
		DataDir: dataDir,
		DHTConfig: dht.ServerConfig{
			StartingNodes: dht.GlobalBootstrapAddrs,
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "creating torrent client")
	}

	return &Client{
		client: torrentClient,
	}, nil
}

func (self *Client) Close() {
	self.torrent.Drop()
	self.client.Close()
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
		Torrent: t,
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

func (self *Client) Render(httpPort int, playAllFiles bool) {
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
			if downloaded >= downloadBuffer {
				fmt.Printf("\nOpen your media player and enter http://127.0.0.1:%d as the network address.\n", httpPort)
			} else {
				fmt.Printf("\nBuffering start of movie. Please wait...\n")
			}
		} else {
			fmt.Printf("\nLoad this M3U playlist into your media player http://127.0.0.1:%d/playlist.m3u\n", httpPort)
		}
	}
}
