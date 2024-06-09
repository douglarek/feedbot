package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/douglarek/feedbot/bot"
	"github.com/douglarek/feedbot/config"
	"github.com/douglarek/feedbot/feed"
	"github.com/gocraft/dbr/v2"
	_ "modernc.org/sqlite"
)

var configFile = flag.String("config-file", "config.jsonc", "path to config file")
var slogLevel = new(slog.LevelVar)

func init() {
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel})
	slog.SetDefault(slog.New(h))
}

func main() {
	flag.Parse()

	settings, err := config.LoadSettings(*configFile)
	if err != nil {
		slog.Error("[main]: cannot load settings", "error", err)
		return

	}
	if settings.EnableDebug {
		slogLevel.Set(slog.LevelDebug)
	}

	db, err := dbr.Open("sqlite", settings.DBFile, nil)
	if err != nil {
		slog.Error("[main]: cannot open database", "error", err)
		return
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	feeder := feed.New(db)
	bot, err := bot.NewDiscordBot(settings.BotToken, feeder)
	if err != nil {
		slog.Error("[main]: cannot create discord bot", "error", err)
		return
	}
	defer bot.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	slog.Info("[main]: bot is running, press Ctrl+C to exit")
	<-stop
	slog.Info("[main]: bot is gracefully shutting down")
}
