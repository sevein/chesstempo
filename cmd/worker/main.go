package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/sevein/chesstempo/game"
)

const usage = `Usage:
    chesstempo-worker [-n NAMESPACE] [-q QUEUE] [-a ADDRESS]
`

func main() {
	log.SetFlags(0)
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "%s\n", usage) }

	var (
		namespaceFlag string
		queueFlag     string
		addressFlag   string
	)

	flag.StringVar(&namespaceFlag, "n", "default", "temporal namespace")
	flag.StringVar(&queueFlag, "q", "queue", "temporal task queue")
	flag.StringVar(&addressFlag, "a", "127.0.0.1:11111", "temporal frontend address")
	flag.Parse()

	ctx := context.Background()

	bot, err := game.NewBot()
	if err != nil {
		log.Fatalln("Unable to create game bot", err)
	}

	c, err := client.NewClient(client.Options{
		Namespace: namespaceFlag,
		HostPort:  addressFlag,
		Logger:    logger{},
	})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, queueFlag, worker.Options{
		DisableWorkflowWorker: true,
	})
	w.RegisterActivityWithOptions(
		game.NewBotActivity(bot).Execute,
		activity.RegisterOptions{Name: game.BotActivityName},
	)

	resp, err := c.WorkflowService().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to server", resp.ServerVersion)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}

type logger struct{}

func (l logger) Debug(msg string, keyvals ...interface{}) {}

func (l logger) Info(msg string, keyvals ...interface{}) {}

func (l logger) Warn(msg string, keyvals ...interface{}) {
	log.Println("Temporal client:", msg, keyvals)
}

func (l logger) Error(msg string, keyvals ...interface{}) {
	log.Println("Temporal client:", msg, keyvals)
}
