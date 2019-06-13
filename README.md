# discord-set-slowmode-bot

Simple discord bot to set channel slowmode to any value between 1 second and 6 hours.

## Why?

I wanted to set the discord slowmode setting to 1 second to not be too restrictive for people who\
talk\
like\
this\
but still try and thwart spambots.

Currently the GUI for setting slowmode is not particularly granular and does not allow for below 5 seconds.

![](https://i.imgur.com/5ki1rDd.png)

I then discovered it's possible to set slowmode to any integer between 0 to 21600 seconds via the discord api, hence the creation of this simple bot.

## How?

1. Create a discord bot app, [instructions here](https://github.com/andersfylling/disgord/wiki/Get-bot-token-and-add-it-to-a-server).
1. Download the [latest release](https://github.com/jfoster/discord-set-slowmode-bot/releases/latest) for your given platform.
1. Open a command line instance in the bot's directory and run ```./discord-set-slowmode-bot```, a warning should be printed ```client id is not specified, check cfg.yaml file``` and cfg.yaml will be created.
1. Copy your bot token into cfg.yaml replacing ```<your-bot-token-here>```.
1. Run ```./discord-set-slowmode-bot``` again, once connected, copy the discord invite link to your favourite browser and add the bot to a server. The bot should now be present in the desired server.  ![](https://transfer.sh/RkNk3/Screenshot-2019-04-02-at-18.09.52.png)
1. In discord, from the channel you would like to set slowmode, type ```@SlowModeBot <ratelimit in seconds>``` e.g. ```@SlowModeBot#8558 1``` for 1 second.  ![](https://i.imgur.com/jPMCDfH.png)
