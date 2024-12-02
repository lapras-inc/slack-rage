package bolt

import (
	"github.com/h3poteto/slack-rage/rage"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"log"
	"os"
)

type Bolt struct {
	channel  string
	verbose  bool
	logger   *logrus.Logger
	detector *rage.Rage
	webApi   *slack.Client
}

func New(threshold, period, speakers int, channel string, verbose bool) *Bolt {
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	slackClient := slack.New(botToken)
	detector := rage.New(threshold, period, speakers, channel, logger, slackClient)
	webApi := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionDebug(verbose),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
	)

	return &Bolt{
		channel,
		verbose,
		logger,
		detector,
		webApi,
	}
}

func (b *Bolt) Start() {
	socketMode := socketmode.New(
		b.webApi,
		socketmode.OptionDebug(b.verbose),
		socketmode.OptionLog(log.New(os.Stdout, "sm: ", log.Lshortfile|log.LstdFlags)),
	)
	authTest, authTestErr := b.webApi.AuthTest()
	if authTestErr != nil {
		b.logger.Errorf("slack bot token is invalid: %v", authTestErr)
		os.Exit(1)
	}
	b.logger.Infof("Bot user ID: %s", authTest.UserID)

	go func() {
		for envelope := range socketMode.Events {
			switch envelope.Type {
			case socketmode.EventTypeConnecting:
				socketMode.Debugf("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				socketMode.Debugf("Connection failed: %v", envelope)
			case socketmode.EventTypeConnected:
				socketMode.Debugf("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := envelope.Data.(slackevents.EventsAPIEvent)
				if !ok {
					socketMode.Debugf("Ignored %+v", envelope)
					continue
				}
				socketMode.Ack(*envelope.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					socketMode.Debugf("CallbackEvent received: %+v", eventsAPIEvent.InnerEvent)
					switch event := eventsAPIEvent.InnerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						err := b.detector.Detect(event.Channel, event.TimeStamp)
						if err != nil {
							socketMode.Debugf("Detect failed: %v", err)
						}
					default:
						socketMode.Debugf("Skipped: %v", event)
					}
				default:
					socketMode.Debugf("Skipped: %v", eventsAPIEvent.Type)
				}
			default:
				socketMode.Debugf("Skipped: %v", envelope.Type)
			}
		}
	}()

	err := socketMode.Run()
	if err != nil {
		b.logger.Errorf("socketMode.Run() failed: %v", err)
		os.Exit(1)
	}
}
