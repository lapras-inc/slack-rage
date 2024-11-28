package bolt

import (
	"github.com/h3poteto/slack-rage/rage"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
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
			case socketmode.EventTypeInteractive:
				callback, ok := envelope.Data.(slack.InteractionCallback)
				if !ok {
					b.logger.Debugf("Ignored %+v", envelope)
					continue
				}
				b.logger.Infof("Interaction received: %+v", callback)
				err := b.detector.Detect(callback.Channel.ID, callback.Message.Text)
				if err != nil {
					b.logger.Errorf("Detect error: %v", err)
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
