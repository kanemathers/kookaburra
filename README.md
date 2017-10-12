# kookaburra

Stream torrents to the media player of your choice

## Installation

    $ go get github.com/kanemathers/kookaburra

## Usage

kookaburra can be used to stream magnet links or torrent files to any media
player capable of viewing network streams.

To stream a video with its magnet link, use the following command:

    $ kookaburra "magnet:?xt=urn:btih:f84b51f0d2c3455ab5dabb6643b4340234cd036e"

Once the torrent is loaded, a list of available files will be presented to
choose from:

    Found these files in the torrent. Select which one you'd like to stream:

        [0] Big_Buck_Bunny_1080p_surround_FrostWire.com.avi
        [1] PROMOTE_YOUR_CONTENT_ON_FROSTWIRE_01_06_09.txt
        [2] Pressrelease_BickBuckBunny_premiere.pdf
        [3] license.txt

In this case, you'd select ``0`` to stream the file ``Big_Buck_Bunny_1080p_surround_FrostWire.com.avi``.

You can then open your media player and enter ``http://127.0.0.1:8080`` as
the URL to stream.
