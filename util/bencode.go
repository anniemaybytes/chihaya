package util

import (
	"bytes"
	"encoding/hex"
	"slices"
	"strconv"
	"time"

	cdb "chihaya/database/types"
)

func bencodeWriteInt64[T ~int64 | ~int](buf *bytes.Buffer, v T) {
	// Static allocation, length of max int64
	var lenBuf [20]byte

	buf.Write(strconv.AppendInt(lenBuf[:0], int64(v), 10))
}

func bencodeWriteString[T ~string | ~[]byte](buf *bytes.Buffer, v T) {
	bencodeWriteInt64(buf, len(v))
	buf.WriteByte(':')
	buf.Write([]byte(v))
}

func bencodeWriteNumber[T ~int64 | ~int](buf *bytes.Buffer, v T) {
	buf.WriteByte('i')
	bencodeWriteInt64(buf, v)
	buf.WriteByte('e')
}

func BencodeFailure(buf *bytes.Buffer, err string, interval time.Duration) {
	if interval < 0 {
		panic("bencode: negative interval")
	}

	buf.WriteByte('d')

	bencodeWriteString(buf, "failure reason")
	bencodeWriteString(buf, err)

	if interval > 0 {
		bencodeWriteString(buf, "interval")
		bencodeWriteNumber(buf, interval/time.Second)
	}

	buf.WriteByte('e')
}

func BencodeSortTorrentHashKeys(keys []cdb.TorrentHash) {
	slices.SortFunc(keys, func(a, b cdb.TorrentHash) int {
		return slices.Compare(a[:], b[:])
	})
}

// BencodeScrapeHeader Writes the scrape header.
// Call BencodeScrapeTorrent afterwards, then finish with BencodeScrapeFooter
func BencodeScrapeHeader(buf *bytes.Buffer) {
	buf.WriteByte('d')

	bencodeWriteString(buf, "files")

	buf.WriteByte('d')
}

func BencodeScrapeTorrent(buf *bytes.Buffer, infoHash cdb.TorrentHash, complete, downloaded, incomplete int64) {
	// Convert to hex inline
	var hashBuf [cdb.TorrentHashSize * 2]byte

	hex.Encode(hashBuf[:], infoHash[:])
	bencodeWriteString(buf, hashBuf[:])

	buf.WriteByte('d')

	bencodeWriteString(buf, "complete")
	bencodeWriteNumber(buf, complete)

	bencodeWriteString(buf, "downloaded")
	bencodeWriteNumber(buf, downloaded)

	bencodeWriteString(buf, "incomplete")
	bencodeWriteNumber(buf, incomplete)

	buf.WriteByte('e')
}

func BencodeScrapeFooter(buf *bytes.Buffer, scrapeInterval int) {
	buf.WriteByte('e')

	bencodeWriteString(buf, "flags")

	buf.WriteByte('d')

	bencodeWriteString(buf, "min_request_interval")
	bencodeWriteNumber(buf, scrapeInterval)

	buf.WriteByte('e')

	buf.WriteByte('e')
}

// BencodeAnnounceHeader Writes the announce header.
// Call BencodeAnnouncePeersIP4 afterwards, then finish with BencodeAnnounceFooter
// TODO: convert interval and minInterval to time.Duration
func BencodeAnnounceHeader(buf *bytes.Buffer, complete, incomplete, downloaded int64, interval, minInterval int) {
	buf.WriteByte('d')

	bencodeWriteString(buf, "complete")
	bencodeWriteNumber(buf, complete)

	bencodeWriteString(buf, "downloaded")
	bencodeWriteNumber(buf, downloaded)

	bencodeWriteString(buf, "incomplete")
	bencodeWriteNumber(buf, incomplete)

	bencodeWriteString(buf, "interval")
	bencodeWriteNumber(buf, interval)

	bencodeWriteString(buf, "min interval")
	bencodeWriteNumber(buf, minInterval)
}

// BencodeAnnouncePeersIP4
// TODO: do not require slice, but has an issue with writing back the number of entries
// TODO: if slice is not needed, we can do a one pass encoding instead of two-pass
func BencodeAnnouncePeersIP4(buf *bytes.Buffer, peers []*cdb.Peer, compact, peerID bool) {
	bencodeWriteString(buf, "peers")

	if compact {
		bencodeWriteInt64(buf, len(peers)*cdb.PeerAddressSize)
		buf.WriteByte(':')

		for _, peer := range peers {
			buf.Write(peer.Addr[:])
		}
	} else {
		buf.WriteByte('l')

		for _, peer := range peers {
			buf.WriteByte('d')

			bencodeWriteString(buf, "ip")
			{
				bencodeWriteInt64(buf, peer.Addr.IPStringLen())
				buf.WriteByte(':')
				peer.Addr.AppendIPString(buf)
			}

			if peerID {
				bencodeWriteString(buf, "peer id")
				bencodeWriteString(buf, peer.ID[:])
			}

			bencodeWriteString(buf, "port")
			bencodeWriteNumber(buf, int64(peer.Addr.Port()))

			buf.WriteByte('e')
		}

		buf.WriteByte('e')
	}
}

func BencodeAnnounceFooter(buf *bytes.Buffer) {
	buf.WriteByte('e')
}
