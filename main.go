package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (fetchItem FetchItem) Title() string {
	if len(fetchItem.ResolvedTitle) > 0 {
		return fetchItem.ResolvedTitle
	}
	return fetchItem.GivenTitle
}

func (fetchItem FetchItem) Url() string {
	if len(fetchItem.ResolvedUrl) > 0 {
		return fetchItem.ResolvedUrl
	}
	return fetchItem.GivenUrl
}

func (fetchItem FetchItem) String() string {
	return fmt.Sprintf("%s ( %s )\n", fetchItem.Title(), fetchItem.Url())
}

func main() {
	consumerKey := os.Getenv("POCKET_CONSUMER_KEY")
	accessToken := os.Getenv("POCKET_ACCESS_TOKEN")
	defaultSlackUrl := os.Getenv("SLACK_POCKET_URL")

	isArchive := flag.Bool("a", false, "archive flag: default false")
	number := flag.Int("n", 10, "item count")
	isSent := flag.Bool("s", false, "send slack flag: default false")
	slackUrl := flag.String("url", defaultSlackUrl, "slack channel url: env value of SLACK_POCKET_URL")

	flag.Parse()

	items, err := getPocketItems(*number, consumerKey, accessToken)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	text := ""
	for _, fetchItem := range items {
		text += fmt.Sprint(fetchItem)

		if *isArchive {
			result, err := archiveItem(fetchItem, consumerKey, accessToken)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println(result)
		}
	}

	if !(*isSent) {
		fmt.Println(text)
		return
	}

	result, err := sendSlack(*slackUrl, text)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(result)
}

func getPocketItems(number int, consumerKey, accessToken string) (list map[int]FetchItem, err error) {
	params := url.Values{}
	params.Set("state", "unread")
	params.Set("sort", "newest")
	params.Set("count", strconv.Itoa(number))
	params.Set("consumer_key", consumerKey)
	params.Set("access_token", accessToken)

	resp, err := http.Get("https://getpocket.com/v3/get?" + params.Encode())
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New(string(body))
		return
	}

	var result FetchResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return
	}

	return result.List, nil
}

func archiveItem(fetchItem FetchItem, consumerKey, accessToken string) (result string, err error) {
	actions := fmt.Sprintf("[{\"action\":\"archive\",\"item_id\":\"%s\",\"time\":\"%d\"}]", fetchItem.ItemId, time.Now().Unix())

	params := url.Values{}
	params.Set("actions", actions)
	params.Set("consumer_key", consumerKey)
	params.Set("access_token", accessToken)

	resp, err := http.Get("https://getpocket.com/v3/send?" + params.Encode())
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New(string(body))
		return
	}

	result = fmt.Sprintf(`Title: "%s" is arhived\n`, fetchItem.Title())
	return
}

func sendSlack(slackUrl, text string) (result string, err error) {
	req, err := http.NewRequest(
		"POST",
		slackUrl,
		bytes.NewBuffer([]byte(`{"text":"`+text+`"}`)))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	result = string(body)
	return
}
