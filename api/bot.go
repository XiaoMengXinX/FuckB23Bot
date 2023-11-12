package main

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type response struct {
	QueryID   string              `json:"inline_query_id"`
	Method    string              `json:"method"`
	Results   []inlineQueryResult `json:"results"`
	CacheTime int64               `json:"cache_time"`
}

type inlineQueryResult struct {
	Type                string              `json:"type"`
	Id                  int64               `json:"id"`
	Title               string              `json:"title"`
	InputMessageContent inputMessageContent `json:"input_message_content"`
	Description         string              `json:"description"`
}

type inputMessageContent struct {
	MessageText string `json:"message_text"`
}

func FuckB23(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)

	var update tgbotapi.Update

	err := json.Unmarshal(body, &update)
	if err != nil {
		return
	}

	if update.InlineQuery != nil {
		messageText := update.InlineQuery.Query
		if messageText == "" {
			return
		}

		originalURLs := ExtractB23URL(messageText)
		var redirectURLs []string
		for i := 0; i < len(originalURLs); i++ {
			var u string
			u, err = GetRedirect(originalURLs[i])
			if err != nil {
				break
			}
			redirectURLs = append(redirectURLs, u)
		}
		replacedText := ReplaceB23URL(messageText, originalURLs, redirectURLs)
		timeNow := time.Now().UnixNano()

		inlineMsgContent := replacedText
		inlineMsgTitle := "Replaced Text"
		if err != nil {
			inlineMsgContent = fmt.Sprintln(err)
			inlineMsgTitle = "Failed to parse"
		}

		data := response{
			Method:    "answerInlineQuery",
			QueryID:   update.InlineQuery.ID,
			CacheTime: 3600,
			Results: []inlineQueryResult{
				{
					Type:                "article",
					Id:                  timeNow,
					Title:               inlineMsgTitle,
					InputMessageContent: inputMessageContent{MessageText: inlineMsgContent},
					Description:         replacedText,
				},
			},
		}
		msg, _ := json.Marshal(data)

		w.Header().Add("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, string(msg))
	}
}

func ExtractB23URL(text string) []string {
	re := regexp.MustCompile(`https?://b23\.tv/[A-Za-z0-9]+`)
	urls := re.FindAllString(text, -1)
	return urls
}

func ReplaceB23URL(text string, oldURLs, newURLs []string) string {
	for i := 0; i < len(oldURLs); i++ {
		text = strings.Replace(text, oldURLs[i], newURLs[i], 1)
	}
	return text
}

func GetRedirect(url string) (redirect string, err error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	redirect = resp.Header.Get("Location")
	if redirect != "" {
		return CleanURL(redirect), nil
	} else {
		return url, fmt.Errorf("Failed to get redirect URL from %s ", url)
	}
}

func CleanURL(originalURL string) string {
	parsedURL, _ := url.Parse(originalURL)
	return fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
}
