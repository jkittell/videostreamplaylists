package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/jkittell/data/api/client"
	"github.com/jkittell/data/database"
	"github.com/jkittell/data/structures"
	"github.com/jkittell/toolbox"
	"github.com/mozillazg/go-slugify"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Playlist struct {
	URL                 string    `bson:"url" json:"url"`
	VOD                 bool      `bson:"vod"`
	MediaSequenceNumber int       `bson:"media_sequence_number" json:"media_sequence_number"`
	Data                string    `bson:"data" json:"data"`
	CreatedAt           time.Time `bson:"created_at" json:"created_at"`
}

func validate(masterPlaylistURL string) bool {
	playlistURLs := getPlaylistURLs(masterPlaylistURL)
	for i := 0; i < playlistURLs.Length(); i++ {
		playlistURL := playlistURLs.Lookup(i)
		p := parsePlaylist(playlistURL)
		if p.VOD {
			return false
		}
		if p.MediaSequenceNumber > 0 {
			return true
		}
	}
	return false
}

func parsePlaylist(url string) Playlist {
	data, err := client.Get(url, nil, nil)
	if err != nil {
		log.Println(err)
	}

	var msn int
	var vod bool

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		// #EXT-X-MEDIA-SEQUENCE:0
		if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE") {
			s := strings.Split(line, ":")
			if len(s) == 2 {
				i, err := strconv.Atoi(s[1])
				if err != nil {
					log.Println(err)
				}
				msn = i
			}
		}
		// #EXT-X-PLAYLIST-TYPE:VOD
		if line == "#EXT-X-PLAYLIST-TYPE:VOD" {
			vod = true
		}
	}

	p := Playlist{
		URL:                 url,
		VOD:                 vod,
		MediaSequenceNumber: msn,
		Data:                string(data),
		CreatedAt:           time.Now(),
	}
	return p
}

func trackLivePlaylist(ctx context.Context, db database.MongoDB[Playlist], masterPlaylistURL string) {
	playlistURLs := make(chan string)
	collectionName := slugify.Slugify(masterPlaylistURL)
	go savePlaylists(ctx, db, collectionName, playlistURLs)
	for range time.Tick(2 * time.Second) {
		arr := getPlaylistURLs(masterPlaylistURL)
		for i := 0; i < arr.Length(); i++ {
			playlistURL := arr.Lookup(i)
			playlistURLs <- playlistURL
		}
	}
}

func savePlaylists(context context.Context, db database.MongoDB[Playlist], collectionName string, playlistURLs chan string) {
	for url := range playlistURLs {
		p := parsePlaylist(url)
		err := db.Insert(context, collectionName, p)
		if err != nil {
			return
		}
	}
}

func getPlaylistURLs(masterPlaylistURL string) *structures.Array[string] {
	playlistURLs := structures.NewArray[string]()
	playlist, err := client.Get(masterPlaylistURL, nil, nil)
	if err != nil {
		log.Println(err)
		return playlistURLs
	}

	baseURL := toolbox.BaseURL(masterPlaylistURL)
	scanner := bufio.NewScanner(bytes.NewReader(playlist))
	for scanner.Scan() {
		var streamURL string
		line := scanner.Text()
		if !strings.Contains(line, "#EXT") && strings.Contains(line, "m3u8") {
			if !strings.Contains(line, "http") {
				streamURL = fmt.Sprintf("%s/%s", baseURL, line)
			} else {
				streamURL = line
			}
			playlistURLs.Push(streamURL)
		} else if strings.Contains(line, "#EXT-X-I-FRAME-STREAM-INF") || strings.Contains(line, "#EXT-X-MEDIA") {
			regEx := regexp.MustCompile("URI=\"(.*?)\"")
			match := regEx.MatchString(line)
			if match {
				s1 := regEx.FindString(line)
				_, s2, _ := strings.Cut(s1, "=")
				s3 := strings.Trim(s2, "\"")
				URI := s3
				if !strings.Contains(line, "http") {
					streamURL = fmt.Sprintf("%s/%s", baseURL, URI)
				} else {
					streamURL = line
				}
				playlistURLs.Push(streamURL)
			}
		}
	}
	return playlistURLs
}
