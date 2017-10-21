# kookaburra

Stream torrents to the media player of your choice.

I have been using [popcorntime](https://popcorntime.sh) for a while for its
torrent streaming abilities but it was very bloated for what I was using it
for. It's an Electron app with a built in torrent browser and media player.
None of which I needed.

I then found [peerflix](https://github.com/mafintosh/peerflix) which is very
similar to kookaburra but it's a Javascript/node.js app. I wasn't interested
in installing node.js for one simple program.

So kookaburra was born. Written in Go, it's a simple, single binary program
to do one thing - stream torrents.

## Installation

    $ go get github.com/kanemathers/kookaburra

## Usage

kookaburra can be used to stream magnet links or torrent files to any media
player capable of viewing network streams.

To stream a movie with its magnet link, use the following command:

    $ kookaburra -largest "magnet:?xt=urn:btih:f84b51f0d2c3455ab5dabb6643b4340234cd036e"

You can then open ``http://127.0.0.1:8080`` in your media player and the movie
will begin streaming.

If the torrent contains multiple files you can omit the ``-largest`` flag to
specify the file you wish to stream. Or, you can pass the ``-all`` flag to
create an M3U playlist of all files to stream (useful for TV series or music
albums).

See ``kookaburra -help`` for more.