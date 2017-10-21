package main

import (
	"net/http"
	"strings"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/pkg/errors"
)

type Client struct {
	client *torrent.Client
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

	return &Torrent{
		Torrent: t,
	}, nil
}
