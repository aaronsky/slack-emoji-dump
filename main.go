package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type slackClient struct {
	token string
}

type emojiResponse struct {
	Ok       bool              `json:"ok"`
	Error    string            `json:"error"`
	Emojis   map[string]string `json:"emoji"`
	Metadata responseMetadata  `json:"response_metadata"`
}

type responseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("TOKEN")
	client := &slackClient{
		token: token,
	}

	emojis, err := client.listEmoji()
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range emojis {
		if strings.HasPrefix(v, "alias:") {
			log.Println("Skipping", k, "because it represents an alias")
			continue
		}

		log.Println("Downloading", k, "to current directory")

		name := k + path.Ext(v)
		err = downloadImageToFile(path.Join(cwd, "emojis", name), v)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (client *slackClient) listEmoji() (emojis map[string]string, err error) {
	emojis = make(map[string]string)
	var limit = 20
	var cursor string

	for {
		url := fmt.Sprintf("https://slack.com/api/emoji.list?token=%s&limit=%d&cursor=%s", client.token, limit, cursor)

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		} else if resp.StatusCode == http.StatusTooManyRequests {
			timeToWait, err := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 64)
			if err != nil {
				return nil, err
			}
			time.Sleep(time.Duration(timeToWait) * time.Second)
			continue
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var e emojiResponse
		err = json.Unmarshal(body, &e)
		if err != nil {
			return nil, err
		} else if !e.Ok {
			return nil, fmt.Errorf(e.Error)
		}

		for k, v := range e.Emojis {
			emojis[k] = v
		}

		if e.Metadata.NextCursor == "" {
			break
		}
		cursor = e.Metadata.NextCursor
	}

	return emojis, err
}

func downloadImageToFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}

	defer out.Close()
	_, err = io.Copy(out, resp.Body)

	return err
}
