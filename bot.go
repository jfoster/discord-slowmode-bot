package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/sirupsen/logrus"
	"github.com/smallfish/simpleyaml"
	"github.com/urfave/cli"
)

const (
	cfgfile         = "cfg.yaml"
	cfgfiletemplate = "cfg.yaml.template"
)

var (
	version = "## filled by go build ##"
	err     error
)

func main() {
	if err := cliApp(); err != nil {
		logrus.Fatal(err)
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

	if err = app.Run(os.Args); err != nil {
		return err
	}
	return nil
}

func getToken(token string) (string, error) {
	if token != "" {
		return token, nil
	}
	if _, err := os.Stat(cfgfile); os.IsNotExist(err) {
		input, err := ioutil.ReadFile(cfgfiletemplate)
		if err != nil {
			return "", err
		}
		err = ioutil.WriteFile(cfgfile, input, 0644)
		if err != nil {
			return "", err
		} else {
			os.Remove(cfgfiletemplate)
		}
	}
	cfg, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return "", err
	} else {
		yaml, err := simpleyaml.NewYaml(cfg)
		if err != nil {
			return "", err
		} else {
			ret, err := yaml.Get("token").String()
			if ret == "" || ret == "<your-bot-token-here>" {
				return "", errors.New("client id is not specified, check " + cfgfile + " file")
			}
			if err != nil {
				return "", err
			}
			return ret, nil
		}
	}
}

func runBot(token string) error {
	logrus.Info("Creating Discord session")
	bot, err := disgord.NewSession(&disgord.Config{
		Token: token,
	})
	if err != nil {
		return err
	}
	bot.On(disgord.EventMessageCreate, onMessageCreate)

	logrus.Info("Discord session created successfully")
	logrus.Info("Starting bot")

	start := time.Now()
	if err = bot.Connect(); err != nil {
		return err
	}
	elapsed := time.Since(start)
	logrus.Info("Connection took ", elapsed)

	me, err := bot.Myself()
	if err != nil {
		return err
	}
	clientID := me.ID.String()
	if len(clientID) > 0 {
		logrus.Info("Link to add the bot to your server:")
		logrus.Info("https://discordapp.com/oauth2/authorize?scope=bot&permissions=16&client_id=" + clientID)
	}

	bot.DisconnectOnInterrupt()
	return nil
}

func onMessageCreate(session disgord.Session, data *disgord.MessageCreate) {
	message := data.Message

	if len(message.Mentions) == 0 {
		return
	}

	me, err := session.Myself()
	if err != nil {
		logrus.Error(err)
	}

	if message.Mentions[0].ID != me.ID {
		return
	}

	channel, err := disgord.GetChannel(session.Req(), message.ChannelID)
	if err != nil {
		logrus.Error(err)
	}

	guild, err := session.GetGuild(channel.GuildID)
	if err != nil {
		logrus.Error(err)
	}

	member, err := session.GetGuildMember(guild.ID, message.Author.ID)
	if err != nil {
		logrus.Error(err)
	}

	if len(member.Roles) == 0 {
		return
	}

	isPermitted := false
	for _, roleID := range member.Roles {
		role, err := guild.Role(roleID)
		if err != nil {
			logrus.Error(err)
		} else if (role.Permissions & (disgord.AdministratorPermission | disgord.ManageChannelsPermission)) != 0 {
			isPermitted = true
			break
		}
	}

	if !isPermitted {
		logrus.Info(member.User.Username + "#" + member.User.Discriminator.String() + " (" + member.Nick + ") is not permitted!")
		return
	}

	contents := strings.Split(message.Content, " ")
	if len(contents) < 2 {
		return
	}

	ratelimit := contents[1]
	if ratelimit == "?" {
		message.RespondString(session, "slowmode is set to "+strconv.FormatUint(uint64(channel.RateLimitPerUser), 10)+" seconds")
	} else {
		u, err := strconv.ParseUint(ratelimit, 10, 0)
		if err != nil {
			logrus.Error(err)
			return
		}
		params := disgord.NewModifyTextChannelParams()
		if err := params.SetRateLimitPerUser(uint(u)); err != nil {
			logrus.Error(err)
			return
		}
		_, err = session.ModifyChannel(channel.ID, params)
		if err != nil {
			logrus.Error(err)
			return
		}
		message.RespondString(session, "slowmode set to "+ratelimit+" seconds")
	}
}
