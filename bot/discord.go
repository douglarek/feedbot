package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/douglarek/feedbot/feed"
)

var (
	defaultMemberPermissions int64 = discordgo.PermissionAdministrator // defaullt to admin only
	discordCommands                = []*discordgo.ApplicationCommand{
		{
			Name:                     "feedbot",
			Description:              "A feed bot",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "list feed subscriptions",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "test",
					Description: "test a feed subscription",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "the url of the feed",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "add a feed subscription",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "the url of the feed",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "remove a feed subscription",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "the url of the feed",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "export",
					Description: "export feed subscriptions to opml file",
				},
			},
		},
	}
	discordRegisteredCommands = make([]*discordgo.ApplicationCommand, len(discordCommands))
)

type Discord struct {
	session *discordgo.Session
}

func (d *Discord) Close() error {
	return d.session.Close()
}

func NewDiscordBot(token string, feeder *feed.Feeder) (*Discord, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	session.AddHandler(discordReady)
	session.AddHandler(discordCommandsHandler(feeder))
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	err = session.Open()
	if err != nil {
		return nil, err
	}

	// create commands
	for i, v := range discordCommands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
		if err != nil {
			return nil, err
		}
		discordRegisteredCommands[i] = cmd
	}

	// auto task
	go checkFeed(session, feeder)

	return &Discord{session: session}, nil
}

func discordReady(_ *discordgo.Session, r *discordgo.Ready) {
	slog.Info("[bot.botReady]: bot is ready", "user", r.User.Username+"#"+r.User.Discriminator)
}

func checkFeed(s *discordgo.Session, f *feed.Feeder) {
	tk := time.NewTicker(1 * time.Minute)
	for t := range tk.C {
		resp, err := f.FindNewItems(context.TODO())
		if err != nil {
			slog.Error("[bot.checkFeed]: error listing feeds", "error", err)
			continue
		}

		if len(resp) == 0 {
			slog.Debug("[bot.checkFeed]: no feeds to check")
			continue
		}

		for _, item := range resp {
			slog.Debug(fmt.Sprintf("[bot.checkFeed]: checking feed [%s]", item.Link), "time", t.Format(time.RFC3339))
			if len(item.NewItems) == 0 {
				slog.Info(fmt.Sprintf("[bot.checkFeed]: no new items for feed [%s]", item.Link))
				continue
			}
			for _, newItem := range item.NewItems {
				slog.Debug(fmt.Sprintf("[bot.checkFeed]: new item for feed [%s]", item.Link), "title", newItem.Title, "link", newItem.Link)
				s.ChannelMessageSend(item.ChannelID, fmt.Sprintf(":newspaper2: New feed from **%s**!\n%s", item.Title, newItem.Link))
			}
		}
	}
}

func discordCommandsHandler(f *feed.Feeder) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.ApplicationCommandData().Name == "feedbot" {
			options := i.ApplicationCommandData().Options
			var (
				content string
				err     error
			)

			switch options[0].Name {
			case "list":
				resp, err := f.List(context.TODO(), &feed.FeederListRequest{ChannelID: i.ChannelID})
				if err == nil {
					content = ":newspaper2: Your subsciptions:\n"
					for _, item := range resp {
						content += fmt.Sprintf("- [%s](%s)\n", item.Title, item.Link)
					}
				}
			case "test":
				resp, serr := f.Fetch(context.TODO(), &feed.FeederFetchRequest{URL: options[0].Options[0].StringValue()})
				err = serr
				if err == nil {
					content = ":newspaper2: Latest feed: " + resp.LatestItemTitle + "\n" + resp.LatestItemLink
				}
			case "add":
				resp, serr := f.Subscribe(context.TODO(), &feed.FeederSubscribeRequest{URL: options[0].Options[0].StringValue(), ChannelID: i.ChannelID})
				err = serr
				if err == nil {
					content = ":newspaper2: Subscribed to [" + resp.Title + "](" + resp.Link + ")"
				}
			case "remove":
				err = f.Unsubscribe(context.TODO(), &feed.FeederUnsubscribeRequest{URL: options[0].Options[0].StringValue(), ChannelID: i.ChannelID})
				if err == nil {
					content = ":newspaper2: Unsubscribed"
				}
			case "export":
				content = "Not implemented yet"
			}

			if err != nil {
				content = ":robot: " + err.Error()
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: content,
				},
			})
		}
	}
}
