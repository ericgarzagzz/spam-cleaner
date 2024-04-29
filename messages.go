package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func ReadMessages(client *http.Client) {
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))

	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	msg_query_result, err := srv.Users.Messages.List("me").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve inbox messages: %v", err)
	}

	for _, msg := range msg_query_result.Messages {
		msg_details, err := srv.Users.Messages.Get("me", msg.Id).Do()

		if err != nil {
			log.Fatalf("Unale to retrieve message %s: %v", msg.Id, err)
		}

		msg_headers := msg_details.Payload.Headers

		subject, err := get_header_value("Subject", msg_headers)

		if err != nil {
			log.Fatalf("Subject header is not present on message %s.", msg.Id)
		}

		from, err := get_header_value("From", msg_headers)

		if err != nil {
			log.Fatalf("From header is not present on message %s.", msg.Id)
		}

		log.Printf("Message: { ID = %s, From = %s, Subject: %s }\n", msg.Id, from, subject)
	}
}

func get_header_value(key string, headers []*gmail.MessagePartHeader) (string, error) {
	idx := slices.IndexFunc(headers, func(header *gmail.MessagePartHeader) bool { return header.Name == key })

	if idx == -1 {
		return "", errors.New("Unable to find a header with the name: " + key)
	}

	return headers[idx].Value, nil
}
