package util

import (
	"bytes"
	"encoding/hex"
	"math"
	"net"
	"slices"
	"testing"
	"time"

	cdb "chihaya/database/types"

	"github.com/zeebo/bencode"
)

var testPeers = []*cdb.Peer{
	{Addr: cdb.NewPeerAddressFromIPPort(net.ParseIP("127.0.0.1"), 12345), ID: cdb.PeerID{1, 2, 3, 4}},
	{Addr: cdb.NewPeerAddressFromIPPort(net.ParseIP("8.8.8.8"), math.MaxInt16), ID: cdb.PeerID{5, 6, 7, 8}},
	{Addr: cdb.NewPeerAddressFromIPPort(net.ParseIP("1.1.10.10"), 22), ID: cdb.PeerID{0, 1, 2, 3, 4, 5}},
}

var testTorrents map[cdb.TorrentHash]*cdb.Torrent

var testTorrentKeys []cdb.TorrentHash

func init() {
	testTorrents = make(map[cdb.TorrentHash]*cdb.Torrent)

	for range 8 {
		t := &cdb.Torrent{}
		t.SeedersLength.Store(UnsafeUint32())
		t.Snatched.Store(UnsafeUint32())
		t.LeechersLength.Store(UnsafeUint32())

		var tKey cdb.TorrentHash
		_, _ = UnsafeReadRand(tKey[:])
		testTorrents[tKey] = t
	}

	testTorrentKeys = make([]cdb.TorrentHash, 0, len(testTorrents))
	for hash := range testTorrents {
		testTorrentKeys = append(testTorrentKeys, hash)
	}
	// pre-sort
	BencodeSortTorrentHashKeys(testTorrentKeys)
}

func testBencodeFailure(t *testing.T, err string, interval time.Duration) {
	buf1 := new(bytes.Buffer)
	marshalerBencodeFailure(buf1, err, interval)

	buf2 := new(bytes.Buffer)
	BencodeFailure(buf2, err, interval)

	if slices.Compare(buf1.Bytes(), buf2.Bytes()) != 0 {
		t.Fatalf("expected \"%s\", got \"%s\"", buf1.Bytes(), buf2.Bytes())
	}
}

func testBencodeScrape(t *testing.T,
	scrapeInterval int,
	torrentKeys []cdb.TorrentHash, torrents map[cdb.TorrentHash]*cdb.Torrent) {
	buf1 := new(bytes.Buffer)
	marshalerBencodeScrape(buf1, scrapeInterval, torrentKeys, torrents)

	buf2 := new(bytes.Buffer)
	BencodeScrapeHeader(buf2)

	for _, k := range torrentKeys {
		t := torrents[k]
		BencodeScrapeTorrent(buf2, k, int64(t.SeedersLength.Load()), int64(t.Snatched.Load()), int64(t.LeechersLength.Load()))
	}

	BencodeScrapeFooter(buf2, scrapeInterval)

	if slices.Compare(buf1.Bytes(), buf2.Bytes()) != 0 {
		t.Fatalf("expected \"%s\", got \"%s\"", buf1.Bytes(), buf2.Bytes())
	}
}

func testBencodeAnnounce(t *testing.T,
	complete, incomplete, downloaded int64,
	interval, minInterval int,
	peers []*cdb.Peer, compact, peerID bool) {
	buf1 := new(bytes.Buffer)
	marshalerBencodeAnnounce(buf1, complete, incomplete, downloaded, interval, minInterval, peers, compact, peerID)

	buf2 := new(bytes.Buffer)
	BencodeAnnounceHeader(buf2, complete, incomplete, downloaded, interval, minInterval)
	BencodeAnnouncePeersIP4(buf2, peers, compact, peerID)
	BencodeAnnounceFooter(buf2)

	if slices.Compare(buf1.Bytes(), buf2.Bytes()) != 0 {
		t.Fatalf("expected \"%s\", got \"%s\"", buf1.Bytes(), buf2.Bytes())
	}
}

func marshalerBencode(buf *bytes.Buffer, data any) error {
	encoder := bencode.NewEncoder(buf)
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func marshalerBencodeFailure(buf *bytes.Buffer, err string, interval time.Duration) {
	data := make(map[string]any)
	data["failure reason"] = err

	if interval > 0 {
		data["interval"] = interval / time.Second // Assuming in seconds
	}

	errx := marshalerBencode(buf, data)
	if errx != nil {
		panic(errx)
	}
}

func marshalerBencodeScrape(buf *bytes.Buffer,
	scrapeInterval int,
	torrentKeys []cdb.TorrentHash, torrents map[cdb.TorrentHash]*cdb.Torrent) {
	data := make(map[string]any)
	data["flags"] = map[string]any{
		"min_request_interval": scrapeInterval,
	}

	files := make(map[string]map[string]any)

	for _, k := range torrentKeys {
		torrent := torrents[k]

		// bug: upstream bencode library doesn't sort keys properly otherwise!
		kk := hex.EncodeToString(k[:])

		files[kk] = map[string]any{
			"complete":   torrent.SeedersLength.Load(),
			"downloaded": torrent.Snatched.Load(),
			"incomplete": torrent.LeechersLength.Load(),
		}
	}

	data["files"] = files

	errx := marshalerBencode(buf, data)
	if errx != nil {
		panic(errx)
	}
}

func marshalerBencodeAnnounce(buf *bytes.Buffer,
	complete, incomplete, downloaded int64,
	interval, minInterval int,
	peers []*cdb.Peer, compact, peerID bool) {
	data := make(map[string]any)
	data["complete"] = complete
	data["incomplete"] = incomplete
	data["downloaded"] = downloaded
	data["interval"] = interval
	data["min interval"] = minInterval

	if compact {
		peerBuff := make([]byte, 0, len(peers)*cdb.PeerAddressSize)

		for _, other := range peers {
			peerBuff = append(peerBuff, other.Addr[:]...)
		}

		data["peers"] = peerBuff
	} else {
		peerList := make([]map[string]any, len(peers))

		for i, other := range peers {
			peerMap := map[string]any{
				"ip":   other.Addr.IPString(),
				"port": other.Addr.Port(),
			}

			if peerID {
				peerMap["peer id"] = other.ID[:]
			}

			peerList[i] = peerMap
		}

		data["peers"] = peerList
	}

	errx := marshalerBencode(buf, data)
	if errx != nil {
		panic(errx)
	}
}

func TestBencode(t *testing.T) {
	t.Run("Failure", func(t *testing.T) {
		testBencodeFailure(t, "test", 0)
		testBencodeFailure(t, "test with interval", 1*time.Hour)
		testBencodeFailure(t, "", 0)
	})

	t.Run("Announce", func(t *testing.T) {
		testBencodeAnnounce(t, 1234, 5678, 9101112, 60, 45, nil, true, false)
		testBencodeAnnounce(t, 1234, 5678, 9101112, 60, 45, nil, false, false)
		testBencodeAnnounce(t, 1234, 5678, 9101112, 60, 45, testPeers, true, false)
		testBencodeAnnounce(t, 1234, 5678, 9101112, 60, 45, testPeers, false, false)
		testBencodeAnnounce(t, 1234, 5678, 9101112, 60, 45, testPeers, false, true)
	})

	t.Run("Scrape", func(t *testing.T) {
		testBencodeScrape(t, 60, testTorrentKeys, testTorrents)
	})
}

func BenchmarkBencode(b *testing.B) {
	b.Run("Failure", func(b *testing.B) {
		b.Run("Native", func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				buf := bytes.NewBuffer(make([]byte, 0, 4096))

				for pb.Next() {
					buf.Reset()
					BencodeFailure(buf, "test with interval", 1*time.Hour)
				}
			})
		})

		b.Run("Marshaler", func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				buf := bytes.NewBuffer(make([]byte, 0, 4096))

				for pb.Next() {
					buf.Reset()
					marshalerBencodeFailure(buf, "test with interval", 1*time.Hour)
				}
			})
		})
	})

	b.Run("Announce", func(b *testing.B) {
		b.Run("Compact", func(b *testing.B) {
			b.Run("Native", func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					buf := bytes.NewBuffer(make([]byte, 0, 4096))

					for pb.Next() {
						buf.Reset()
						BencodeAnnounceHeader(buf, 1234, 5678, 9101112, 60, 45)
						BencodeAnnouncePeersIP4(buf, testPeers, true, false)
						BencodeAnnounceFooter(buf)
					}
				})
			})

			b.Run("Marshaler", func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					buf := bytes.NewBuffer(make([]byte, 0, 4096))

					for pb.Next() {
						buf.Reset()
						marshalerBencodeAnnounce(buf, 1234, 5678, 9101112, 60, 45, testPeers, true, false)
					}
				})
			})
		})
		b.Run("Default", func(b *testing.B) {
			b.Run("Native", func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					buf := bytes.NewBuffer(make([]byte, 0, 4096))

					for pb.Next() {
						buf.Reset()
						BencodeAnnounceHeader(buf, 1234, 5678, 9101112, 60, 45)
						BencodeAnnouncePeersIP4(buf, testPeers, false, false)
						BencodeAnnounceFooter(buf)
					}
				})
			})

			b.Run("Marshaler", func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					buf := bytes.NewBuffer(make([]byte, 0, 4096))

					for pb.Next() {
						buf.Reset()
						marshalerBencodeAnnounce(buf, 1234, 5678, 9101112, 60, 45, testPeers, false, false)
					}
				})
			})
		})
	})

	b.Run("Scrape", func(b *testing.B) {
		b.Run("Native", func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				buf := bytes.NewBuffer(make([]byte, 0, 4096))

				for pb.Next() {
					buf.Reset()
					BencodeScrapeHeader(buf)

					for _, k := range testTorrentKeys {
						t := testTorrents[k]
						BencodeScrapeTorrent(buf, k,
							int64(t.SeedersLength.Load()),
							int64(t.Snatched.Load()),
							int64(t.LeechersLength.Load()),
						)
					}

					BencodeScrapeFooter(buf, 60)
				}
			})
		})

		b.Run("Marshaler", func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				buf := bytes.NewBuffer(make([]byte, 0, 4096))

				for pb.Next() {
					buf.Reset()
					marshalerBencodeScrape(buf, 60, testTorrentKeys, testTorrents)
				}
			})
		})
	})
}
