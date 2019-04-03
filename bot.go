package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Gurpartap/logrus-stack"
	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/event"
	"github.com/andersfylling/disgord/std"
	"github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

const (
	cfgfilename     = "cfg.yaml"
	cfgfiletemplate = `token: <your-bot-token-here>`
)

var (
	err error

	logr = logrus.New()

	DEBUG   = false
	VERSION = "## filled by go build ##"
)

func init() {
	formatter := &logrus.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	}
	DEBUG, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	if DEBUG {
		logr.AddHook(logrus_stack.StandardHook())
		logr.SetLevel(logrus.DebugLevel)
		logr.SetFormatter(&runtime.Formatter{
			ChildFormatter: formatter,
			Line:           true,
			Package:        true,
			File:           true,
		})
	} else {
		logr.SetFormatter(formatter)
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
	app.Version = VERSION
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
		token := c.String("token")
		// get token from cfg if not specified with flag
		if token == "" {
			cfg, err := getCfg()
			if err != nil {
				return err
			}

			token = cfg.GetString("token")
		}

		if token == "" || token == "<your-bot-token-here>" {
			return errors.New(fmt.Sprintf("no bot token specified, edit %s or specify with --token flag", cfgfilename))
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

func getCfg() (*viper.Viper, error) {
	cfg := viper.New()

	cfgfilepath, err := getCfgFilePath()
	if err != nil {
		return cfg, err
	}

	parts := strings.Split(cfgfilename, ".")
	cfg.SetConfigName(parts[0])
	cfg.SetConfigType(parts[1])

	cfg.AddConfigPath(".")
	cfg.AddConfigPath(cfgfilepath)

	if err := cfg.ReadInConfig(); err != nil {
		if _, notFound := err.(viper.ConfigFileNotFoundError); notFound {
			// create dir
			err = os.MkdirAll(cfgfilepath, os.ModePerm)
			if err != nil {
				return cfg, err
			}

			// create file
			file, err := os.Create(filepath.Join(cfgfilepath, filepath.Base("cfg.yaml")))
			if err != nil {
				return cfg, err
			}
			if err = file.Close(); err != nil {
				return cfg, err
			}

			// read cfg template
			err = cfg.ReadConfig(bytes.NewBuffer([]byte(cfgfiletemplate)))
			if err != nil {
				return cfg, err
			}

			logr.Infof("%s created at %s, please copy your bot token into this file", cfgfilename, cfgfilepath)

			// write cfg template
			err = cfg.WriteConfig()
			if err != nil {
				return cfg, err
			}
		}
		return cfg, err
	}

	return cfg, nil
}

func getCfgFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "discordsetslowmodebot"), nil
}

func runBot(token string) error {
	logr.Info("Creating Discord session")

	config := &disgord.Config{
		BotToken: token,
	}
	if DEBUG {
		config.Logger = logr
	}

	bot, err := disgord.NewClient(config)
	if err != nil {
		return err
	}

	filter, err := std.NewMsgFilter(bot)
	if err != nil {
		return err
	}
	bot.On(event.MessageCreate, filter.HasBotMentionPrefix, onMessageCreate)
	bot.On(event.Ready, onReady)

	bot.AddPermission(disgord.PermissionManageChannels)
	url, err := bot.CreateBotURL()
	if err != nil {
		return err
	}

	logr.Info("Starting bot connection")
	start := time.Now()
	if err = bot.Connect(); err != nil {
		return err
	}
	logr.Info("Connection took ", time.Since(start))
	logr.Infof("Link to add the bot to your server:\n%s", url)

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
	guildString := "Connected guilds:\n"
	for i, guild := range guilds {
		guildString += fmt.Sprintf("%d: %s (%s)\n", i+1, guild.Name, guild.ID)
	}
	logr.Info(guildString)
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
