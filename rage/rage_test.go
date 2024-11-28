package rage

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slacktest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

func TestRage_Detect(t *testing.T) {
	ast := assert.New(t)

	conversationsHistoryCallCount := 0
	userInfoCallCount := 0
	conversationsListCallCount := 0
	chatPostMessageCallCount := 0

	ts := slacktest.NewTestServer(func(c slacktest.Customize) {
		c.Handle("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
			conversationsHistoryCallCount++
			response := slack.GetConversationHistoryResponse{
				Messages: []slack.Message{
					{
						Msg: slack.Msg{
							Text:      "userA message1",
							Timestamp: "1234567890",
							User:      "U12345678",
						},
					},
					{
						Msg: slack.Msg{
							Text:      "userA message2",
							Timestamp: "1234567891",
							User:      "U12345678",
						},
					},
					{
						Msg: slack.Msg{
							Text:      "userB message1",
							Timestamp: "1234567892",
							User:      "U12345679",
						},
					},
					{
						Msg: slack.Msg{
							Text:      "userC message1",
							Timestamp: "1234567893",
							User:      "U12345680",
						},
					},
					{
						Msg: slack.Msg{
							Text:      "bot message",
							Timestamp: "1234567894",
							User:      "B12345678",
						},
					},
				},
			}
			bs, err := json.Marshal(response)
			ast.NoError(err)

			_, err = w.Write(bs)
			ast.NoError(err)
		})
		c.Handle("/users.info", func(w http.ResponseWriter, r *http.Request) {
			userInfoCallCount++
			userId := r.FormValue("user")

			isBot := false
			if strings.HasPrefix(userId, "B") {
				isBot = true
			}
			println(userId, isBot)

			user := slack.User{
				ID:    userId,
				IsBot: isBot,
			}
			bs, err := json.Marshal(struct {
				User slack.User `json:"user"`
			}{
				User: user,
			})
			ast.NoError(err)

			_, err = w.Write(bs)
			ast.NoError(err)
		})
		c.Handle("/conversations.list", func(w http.ResponseWriter, r *http.Request) {
			conversationsListCallCount++
			channels := []slack.Channel{
				{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C12345678",
						},
						Name: "random",
					},
				},
			}
			bs, err := json.Marshal(struct {
				Channels []slack.Channel `json:"channels"`
			}{
				Channels: channels,
			})
			ast.NoError(err)

			_, err = w.Write(bs)
			ast.NoError(err)
		})
		c.Handle("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
			chatPostMessageCallCount++
			channel := r.FormValue("channel")
			text := r.FormValue("text")

			ast.Equal("C12345678", channel)
			ast.Equal("<#test-channel> が盛り上がってるっぽいよ！", text)

			_, err := w.Write([]byte("{}"))
			ast.NoError(err)
		})
	})
	ts.Start()
	defer ts.Stop()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	slackClient := slack.New("test-token", slack.OptionAPIURL(ts.GetAPIURL()))
	rage := New(10, 1200, 3, "random", logger, slackClient)

	err := rage.Detect("test-channel", "1234567900")
	ast.NoError(err)
	ast.Equal(1, conversationsHistoryCallCount)
	ast.Equal(4, userInfoCallCount)
	ast.Equal(1, conversationsListCallCount)
	ast.Equal(1, chatPostMessageCallCount)
}

func TestRage_UserIsBot(t *testing.T) {
	ast := assert.New(t)

	userInfoCallCount := 0

	ts := slacktest.NewTestServer(func(c slacktest.Customize) {
		c.Handle("/users.info", func(w http.ResponseWriter, r *http.Request) {
			userInfoCallCount++
			userId := r.FormValue("user")

			isBot := false
			if strings.HasPrefix(userId, "B") {
				isBot = true
			}
			println(userId, isBot)

			user := slack.User{
				ID:    userId,
				IsBot: isBot,
			}
			bs, err := json.Marshal(struct {
				User slack.User `json:"user"`
			}{
				User: user,
			})
			ast.NoError(err)

			_, err = w.Write(bs)
			ast.NoError(err)
		})
	})
	ts.Start()
	defer ts.Stop()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	slackClient := slack.New("test-token", slack.OptionAPIURL(ts.GetAPIURL()))
	rage := New(10, 1200, 3, "random", logger, slackClient)

	isBot, err := rage.UserIsBot("U12345678")
	ast.NoError(err)
	ast.False(isBot)

	isBot, err = rage.UserIsBot("B12345678")
	ast.NoError(err)
	ast.True(isBot)

	ast.Equal(2, userInfoCallCount)
}
