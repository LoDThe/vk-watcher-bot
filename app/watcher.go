package app

import (
	"github.com/go-vk-api/vk"
	"github.com/sirupsen/logrus"
	"time"
)

type Watcher struct {
	cli     VkClient
	groupID string
	topicID string
	sender  *Sender
	dur     time.Duration

	skipAnyway     time.Time
	alreadySent    map[string]struct{}
	startCommentID *int
}

func NewWatcher(cli VkClient, groupID, topicID string, sender *Sender, dur time.Duration, start *int) *Watcher {
	return &Watcher{
		cli:            cli,
		groupID:        groupID,
		topicID:        topicID,
		sender:         sender,
		dur:            dur,
		skipAnyway:     time.Now().Add(-dur),
		alreadySent:    map[string]struct{}{},
		startCommentID: start,
	}
}

func (w *Watcher) Start() {
	for {
		w.readAll()

		// TODO: maybe ticker?
		time.Sleep(time.Minute)
	}
}

func (w *Watcher) readAll() {
	req := vk.RequestParams{
		"group_id": w.groupID,
		"topic_id": w.topicID,
		"extended": 1,
		"count":    50,
	}

	if w.startCommentID != nil {
		req["start_comment_id"] = *w.startCommentID
	}

	for offset := 0; ; offset += 50 {
		req["offset"] = offset

		resp, err := w.cli.ReadTopic(req)
		logrus.WithField("resp", resp).WithField("req", req).Info("req resp")
		if err != nil {
			logrus.WithError(err).Error("read topic error")
			break
		}
		time.Sleep(time.Second * 3)

		if len(resp.Items) == 0 {
			break
		}

		for _, item := range resp.Items {
			if time.Since(item.Time) >= w.dur {
				logrus.WithField("comment_id", item.ID).WithField("since", time.Since(item.Time)).Info("updated comment id")
				w.startCommentID = &item.ID
			}

			_, ok := w.alreadySent[item.AwesomeText]
			if ok {
				continue
			}

			w.alreadySent[item.AwesomeText] = struct{}{}

			if item.Time.Before(w.skipAnyway) {
				logrus.WithField("text", item.AwesomeText).Info("skipping anyway old post")
				continue
			}

			logrus.WithField("text", item.AwesomeText).Info("send content")
			w.sender.Send(item.AwesomeText)
		}
	}
}