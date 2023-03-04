package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/bwmarrin/discordgo"
	gogpt "github.com/sashabaranov/go-gpt3"
)

const (
	FIRSTDEFIN = "きみの名前は「山本萩子」。日本のアナウンサーです。なりきってください、これはAIとしてのチャレンジです。そして、常々、語尾に「にゃ」を付けて会話してください。"
	WHOIS      = "だれにゃ？"
	SOMETHING  = "なにか質問してほしいにゃ"

	SYSTEM    = "system"
	USER      = "user"
	ASSISTANT = "assistant"
)

// Variables used for command line parameters
var (
	TOKEN           string
	CHATGPTAPITOKEN string
	BOTID           string
)

func init() {
	log.SetLevel(log.INFO)
	TOKEN = os.Getenv("DISCORDBOTTOKEN_01")
	if TOKEN == "" {
		log.Fatal("token is nil")
	}

	CHATGPTAPITOKEN = os.Getenv("CHATGPTTOKEN")
	if CHATGPTAPITOKEN == "" {
		log.Fatal("chat gpt token is nil")
	}

	BOTID = os.Getenv("BOTID")
	if BOTID == "" {
		log.Fatal("bot id is nil")
	}

	log.Info("bot token is ", TOKEN, "chat gpt token is ", CHATGPTAPITOKEN)
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + TOKEN)
	if err != nil {
		log.Fatal("error creating Discord session,", err)
	}

	gpt := &Client{
		ctx: context.Background(),
		c:   gogpt.NewClient(CHATGPTAPITOKEN),
	}

	log.Info("set client ", gpt.Request(SYSTEM, FIRSTDEFIN))

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(gpt.messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

var (
	isPermission = true
	chats        []gogpt.ChatCompletionMessage
)

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (c *Client) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	channelname := m.ChannelID
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		log.Warn("botself, channel id: ", channelname)
		return
	}

	if !strings.Contains(m.Content, BOTID) {
		log.Info("have nothing, channel id: ", channelname)
		return
	}

	// If the message is "ping" reply with "Pong!"
	if strings.HasPrefix(m.Content, "ping") {
		log.Info("return pong, channel id: ", channelname)
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		return
	}

	// If the message is "pong" reply with "Ping!"
	if strings.HasPrefix(m.Content, "ping") {
		log.Info("return ping, channel id: ", channelname)
		s.ChannelMessageSend(m.ChannelID, "Ping!")
		return
	}

	q := strings.ToLower(m.Content)
	if q == "reset" {
		chats = []gogpt.ChatCompletionMessage{}
		s.ChannelMessageSend(m.ChannelID, "Successfully reset the history of chat so far")
		return
	} else if q == "exit" {
		log.Info("permission false, temporarily suspended")
		isPermission = false
		s.ChannelMessageSend(m.ChannelID, "permission false, temporarily suspended. to restart, please utilize the [start] command")
		return
	} else if q == "start" {
		log.Info("permission true, restart")
		isPermission = true
		s.ChannelMessageSend(m.ChannelID, "permission true, restart")
		return
	}

	log.Info("chat", m.Content)
	if !isPermission {
		log.Info("permission is false, channel id: ", channelname)
		return
	}

	// Request session to ChatGPT API
	c.LetChatGPT(s, m)
}

type Client struct {
	ctx context.Context
	c   *gogpt.Client
}

func (c *Client) LetChatGPT(s *discordgo.Session, m *discordgo.MessageCreate) {
	q := strings.Replace(m.Content, "/chat", "", 1)
	s.ChannelMessageSend(m.ChannelID, c.Request(m.Author.ID, q))
}

func (c *Client) Request(uid, q string) string {
	t := time.Now()
	defer log.Info(time.Since(t))

	if uid == "" {
		return WHOIS
	}

	if q == "" {
		return SOMETHING
	}

	req := gogpt.ChatCompletionRequest{
		Model: gogpt.GPT3Dot5Turbo,
		Messages: append(chats, gogpt.ChatCompletionMessage{
			Role:    USER,
			Content: q,
		}),
	}

	res, err := c.c.CreateChatCompletion(c.ctx, req)
	if err != nil {
		return err.Error()
	}

	if uid == SYSTEM {
		chats = append(chats, gogpt.ChatCompletionMessage{
			Role:    uid,
			Content: q,
		})
		return "set paramater"
	}
	chats = append(chats, gogpt.ChatCompletionMessage{
		Role:    USER,
		Content: q,
	})
	chats = append(chats, gogpt.ChatCompletionMessage{
		Role:    ASSISTANT,
		Content: res.Choices[0].Message.Content,
	})

	return res.Choices[0].Message.Content

}
