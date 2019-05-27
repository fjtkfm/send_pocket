package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type FetchResult struct {
	List map[int]FetchItem
}

type FetchItem struct {
	ItemId                 string                       `json:"item_id"`
	ResolvedId             string                       `json:"resolved_id"`
	GivenUrl               string                       `json:"given_url"`
	GivenTitle             string                       `json:"given_title"`
	Favorite               string                       `json:"favorite"`
	Status                 string                       `json:"status"`
	TimeAdded              string                       `json:"time_added"`
	TimeUpdated            string                       `json:"time_updated"`
	TimeRead               string                       `json:"time_read"`
	TimeFavorited          string                       `json:"time_favorited"`
	SortId                 int                          `json:"sort_id"`
	ResolvedTitle          string                       `json:"resolved_title"`
	ResolvedUrl            string                       `json:"resolved_url"`
	Excerpt                string                       `json:"excerpt"`
	IsArticle              string                       `json:"is_article"`
	IsIndex                string                       `json:"is_index"`
	HasVideo               string                       `json:"has_video"`
	HasImage               string                       `json:"has_image"`
	WordCount              string                       `json:"word_count"`
	Lang                   string                       `json:"lang"`
	TopImageUrl            string                       `json:"top_image_url"`
	ListenDurationEstimate int                          `json:"listen_duration_estimate"`
	AmpUrl                 string                       `json:"amp_url"`
	DomainMetadata         map[string]string            `json:"domain_metadata"`
	Tags                   map[string]map[string]string `json:"tags"`
}

func main() {
	consumerKey := os.Getenv("POCKET_CONSUMER_KEY")
	accessToken := os.Getenv("POCKET_ACCESS_TOKEN")
	defaultSlackUrl := os.Getenv("SLACK_POCKET_URL")

	isArchive := flag.Bool("a", false, "archive flag: default false")
	number := flag.Int("n", 10, "item count")
	sendSlack := flag.Bool("s", false, "send slack flag: default false")
	slackUrl := flag.String("url", defaultSlackUrl, "slack channel url: env value of SLACK_POCKET_URL")

	flag.Parse()

	params := url.Values{}
	params.Set("state", "unread")
	params.Set("sort", "newest")
	params.Set("count", strconv.Itoa(*number))
	params.Set("consumer_key", consumerKey)
	params.Set("access_token", accessToken)

	resp, err := http.Get("https://getpocket.com/v3/get?" + params.Encode())
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(string(body))
		return
	}

	var result FetchResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	text := ""
	for _, fetchItem := range result.List {
		var title string
		if len(fetchItem.ResolvedTitle) > 0 {
			title = fetchItem.ResolvedTitle
		} else {
			title = fetchItem.GivenTitle
		}
		var itemUrl string
		if len(fetchItem.ResolvedUrl) > 0 {
			itemUrl = fetchItem.ResolvedUrl
		} else {
			itemUrl = fetchItem.GivenUrl
		}
		text += fmt.Sprintf("%s ( %s )\n", title, itemUrl)

		if *isArchive {
			actions := fmt.Sprintf("[{\"action\":\"archive\",\"item_id\":\"%s\",\"time\":\"%d\"}]", fetchItem.ItemId, time.Now().Unix())

			params := url.Values{}
			params.Set("actions", actions)
			params.Set("consumer_key", consumerKey)
			params.Set("access_token", accessToken)

			resp, err := http.Get("https://getpocket.com/v3/send?" + params.Encode())
			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			if resp.StatusCode != http.StatusOK {
				fmt.Println(string(body))
				continue
			}

			fmt.Printf(`Title: "%s" is arhived\n`, title)
		}
	}

	if !(*sendSlack) {
		fmt.Println(text)
		return
	}

	req, err := http.NewRequest(
		"POST",
		*slackUrl,
		bytes.NewBuffer([]byte(`{"text":"`+text+`"}`)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer res.Body.Close()

	fmt.Println(res)
}
