package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/event"
	"github.com/andersfylling/disgord/std"
	"github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/sirupsen/logrus"
	"github.com/smallfish/simpleyaml"
	"github.com/urfave/cli"
)

const (
	cfgfile         = "cfg.yaml"
	cfgfiletemplate = "cfg.yaml.template"
)

var (
	cfg  = new(simpleyaml.Yaml)
	logr = logrus.New()

	version = "## filled by go build ##"
)

func init() {
	cfg, _ = getCfg()
	debug, _ := cfg.Get("debug").Bool()
	if debug {
		logr.SetFormatter(&runtime.Formatter{
			ChildFormatter: &logrus.TextFormatter{
				ForceColors: true,
			},
			Line:    true,
			Package: true,
			File:    true,
		})
	}
}

func main() {
	if err := cliApp(); err != nil {
		logr.Fatal(err)
	}
}

func cliApp() error {
	app := cli.NewApp()
	app.Name = "discord-set-slowmode-bot"
	app.Usage = ""
	app.Version = version
	app.Authors = []cli.Author{
		{
			Name:  "Jacob Foster",
			Email: "jacobfoster@pm.me",
		},
	}
	app.Compiled = time.Now()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "token, t",
			Usage: "Specify bot token.",
		},
	}
	app.Action = func(c *cli.Context) error {
		token, err := getToken(c.String("token"))
		if err != nil {
			return err
		}
		if err := runBot(token); err != nil {
			return err
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		return err
	}
	return nil
}

func getCfg() (*simpleyaml.Yaml, error) {
	yaml := new(simpleyaml.Yaml)

	if _, err := os.Stat(cfgfile); os.IsNotExist(err) {
		input, err := ioutil.ReadFile(cfgfiletemplate)
		if err != nil {
			return yaml, err
		}
		err = ioutil.WriteFile(cfgfile, input, 0644)
		if err != nil {
			return yaml, err
		} else {
			os.Remove(cfgfiletemplate)
		}
	}

	file, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return yaml, err
	}

	yaml, err = simpleyaml.NewYaml(file)
	if err != nil {
		return yaml, err
	}
	return yaml, nil
}

func getToken(token string) (string, error) {
	if token != "" {
		return token, nil
	}

	t, err := cfg.Get("token").String()
	if err != nil {
		return "", err
	}

	if t == "" || t == "<your-bot-token-here>" {
		return "", errors.New("client id is not specified, check " + cfgfile + " file")
	}
	return t, nil
}

func runBot(token string) error {
	logr.Info("Creating Discord session")

	bot, err := disgord.NewClient(&disgord.Config{
		BotToken: token,
		Logger:   logr,
	})
	if err != nil {
		return err
	}

	filter, err := std.NewMsgFilter(bot)
	if err != nil {
		return err
	}

	bot.On(event.Ready, onReady)
	bot.On(event.MessageCreate, filter.HasBotMentionPrefix, onMessageCreate)

	logr.Info("Discord session created successfully")
	logr.Info("Starting bot")

	start := time.Now()
	if err = bot.Connect(); err != nil {
		return err
	}
	logr.Info("Connection took ", time.Since(start))

	bot.AddPermission(disgord.PermissionManageChannels)
	url, err := bot.CreateBotURL()
	if err != nil {
		return err
	}
	logr.Infof("Link to add the bot to your server: %s", url)

	bot.DisconnectOnInterrupt()
	return nil
}

func onReady(session disgord.Session, evt *disgord.Ready) {
	guilds, err := session.GetGuilds(&disgord.GetCurrentUserGuildsParams{
		Limit: 999,
	})
	if err != nil {
		logr.Error(err)
		return
	}
	for i, guild := range guilds {
		logr.Infof("%d: %s", i+1, guild.Name)
	}
}

func onMessageCreate(session disgord.Session, evt *disgord.MessageCreate) {
	message := evt.Message

	channel, err := session.GetChannel(message.ChannelID)
	if err != nil {
		logr.Error(err)
		return
	}

	guild, err := session.GetGuild(channel.GuildID)
	if err != nil {
		logr.Error(err)
		return
	}

	member, err := session.GetMember(guild.ID, message.Author.ID)
	if err != nil {
		logr.Error(err)
		return
	}

	isPermitted := false
	for _, roleID := range member.Roles {
		role, err := guild.Role(roleID)
		if err != nil {
			logr.Error(err)
		} else if (role.Permissions & (disgord.PermissionAdministrator | disgord.PermissionManageChannels)) != 0 {
			isPermitted = true
			break
		}
	}
	if guild.OwnerID == member.User.ID {
		isPermitted = true
	}
	if !isPermitted {
		logr.Infof("%s#%s (%s) is not permitted!", member.User.Username, member.User.Discriminator.String(), member.Nick)
		return
	}

	contents := strings.Split(message.Content, " ")
	if len(contents) < 2 {
		return
	}

	ratelimit := contents[1]
	if ratelimit == "?" {
		message.Reply(session, fmt.Sprintf("slowmode is set to %s seconds", strconv.FormatUint(uint64(channel.RateLimitPerUser), 10)))
	} else {
		u, err := strconv.ParseUint(ratelimit, 10, 0)
		if err != nil {
			logr.Error(err)
			return
		}
		session.UpdateChannel(channel.ID).SetRateLimitPerUser(uint(u)).Execute()

		message.Reply(session, fmt.Sprintf("slowmode set to %s seconds", ratelimit))
	}
}
