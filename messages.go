package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type MessageHeaderSummary struct {
	ID                  string
	Subject             string
	From                string
	To                  string
	UnsubscribeLink     string
	UnsubscribeLinkType UnsubscribeLinkType
}

type UnsubscribeLinkType int

const (
	Url UnsubscribeLinkType = iota + 1
	Mailto
	UrlOneClick
	None
)

func (s *MessageHeaderSummary) String() string {
	return fmt.Sprintf("{ ID = %s, From = %s, Subject = %s }", s.ID, s.From, s.Subject)
}

func (s *MessageHeaderSummary) MarkdownString() string {
	var url string = s.UnsubscribeLink
	if s.UnsubscribeLinkType == UrlOneClick || s.UnsubscribeLinkType == Url {
		var re = regexp.MustCompile(`\b(?:https?://)[^\s<>\[\]]+\b`)

		if len(re.FindStringIndex(url)) > 0 {
			url = re.FindString(url)
		}
	}
	return fmt.Sprintf("- [ ] [%s](%s)", s.From, url)
}

func ReadMessages(client *http.Client) []*MessageHeaderSummary {
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))

	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	msg_query_result, err := srv.Users.Messages.List("me").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve inbox messages: %v", err)
	}

	results := make([]*MessageHeaderSummary, 0, len(msg_query_result.Messages))

	for _, msg := range msg_query_result.Messages {
		msg_details, err := srv.Users.Messages.Get("me", msg.Id).Do()

		if err != nil {
			log.Fatalf("Unale to retrieve message %s: %v", msg.Id, err)
		}

		header_summary, err := get_header_summary(msg_details)

		if err != nil {
			log.Fatalf("Unable to read email headers: %v", err)
		}

		results = append(results, header_summary)
	}

	return results
}

func get_header_summary(message *gmail.Message) (*MessageHeaderSummary, error) {

	headers := message.Payload.Headers

	subject, err := get_header_value("Subject", headers)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Subject header is not present on message %s.", message.Id))
	}

	from, err := get_header_value("From", headers)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("From header is not present on message %s.", message.Id))
	}

	to, err := get_header_value("To", headers)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("To header is not present on message %s.", message.Id))
	}

	unsub_link, err := get_header_value("List-Unsubscribe", headers)
	unsub_link_type := get_unsubscribe_link_type(headers)

	return &MessageHeaderSummary{
		ID:                  message.Id,
		Subject:             subject,
		From:                from,
		To:                  to,
		UnsubscribeLink:     unsub_link,
		UnsubscribeLinkType: unsub_link_type,
	}, nil
}

func get_header_value(key string, headers []*gmail.MessagePartHeader) (string, error) {
	idx := slices.IndexFunc(headers, func(header *gmail.MessagePartHeader) bool {
		return header.Name == key
	})

	if idx == -1 {
		return "", errors.New("Unable to find a header with the name: " + key)
	}

	return headers[idx].Value, nil
}

func get_unsubscribe_link_type(headers []*gmail.MessagePartHeader) UnsubscribeLinkType {
	link, err := get_header_value("List-Unsubscribe", headers)

	if err != nil {
		return None
	}

	unsub_post, _ := get_header_value("List-Unsubscribe-Post", headers)
	var one_click bool = false

	if unsub_post == "List-Unsubscribe=One-Click" {
		one_click = true
	}

	url_idx := strings.Index(link, "<http")
	mailto_idx := strings.Index(link, "<mailto:")

	if url_idx != -1 && one_click {
		return UrlOneClick
	} else if url_idx != -1 {
		return Url
	} else if mailto_idx != -1 {
		return Mailto
	}

	return None
}
