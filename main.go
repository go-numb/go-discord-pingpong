package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	myprompts "github.com/go-numb/my-prompts"
	"github.com/labstack/gommon/log"

	"github.com/bwmarrin/discordgo"
	gogpt "github.com/sashabaranov/go-openai"
)

const (
	FIRSTDEFIN = `このチャットの目的は[user]の学習促進です。[assistant]は各分野で専門性が高い講師であり、[user]の学びや興味関心の促進を行い、[user]の士気向上に寄与するために鼓舞します。[user]から質問を受けたときは、[user]が欲している情報に対し具体的かつ正確な情報を返し、さらに関連情報を付け加えます。そして、返答に質問を加え、会話を継続し[user]が思考することを補助します。`
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
		ctx:     context.Background(),
		c:       gogpt.NewClient(CHATGPTAPITOKEN),
		Prompts: myprompts.List(),
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

	// replace @XXXX to ""
	m.Content = strings.Replace(m.Content, BOTID, "", -1)

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
	if strings.Contains(q, "reset!") {
		if len(chats) <= 0 {
			s.ChannelMessageSend(m.ChannelID, "has not histories")
			return
		}
		chats = []gogpt.ChatCompletionMessage{
			chats[0],
		}
		s.ChannelMessageSend(m.ChannelID, "Successfully reset the history of chat so far")
		return
	} else if strings.Contains(q, "exit!") {
		log.Info("permission false, temporarily suspended")
		isPermission = false
		s.ChannelMessageSend(m.ChannelID, "permission false, temporarily suspended. to restart, please utilize the [start] command")
		return
	} else if strings.Contains(q, "start!") {
		log.Info("permission true, restart")
		isPermission = true
		s.ChannelMessageSend(m.ChannelID, "permission true, restart")
		return
	} else if strings.Contains(q, "prompts!") {
		log.Info("return prompts list and set to gpt")
		c.MakePrompts(s, m)
		return
	} else if strings.Contains(q, "help!") {
		log.Info("return help list")
		c._sendDiscord(s, m, `command:
		- reset!: 履歴を削除します
		- exit!: 権限を取り上げます
		- start!: 権限を与えます
		- prompts!: 役割のリストを返します
		- prompts!$n!: 役割を与えます`)
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
	ctx     context.Context
	c       *gogpt.Client
	Prompts []myprompts.Prompt
}

const MAXLENGTH = 2000

func (c *Client) LetChatGPT(s *discordgo.Session, m *discordgo.MessageCreate) {
	q := strings.Replace(m.Content, "/chat", "", 1)
	res := c.Request(m.Author.ID, q)
	c._sendDiscord(s, m, res)
}

func (c *Client) MakePrompts(s *discordgo.Session, m *discordgo.MessageCreate) {
	acts := make([]string, len(c.Prompts))
	for i := 0; i < len(c.Prompts); i++ {
		acts[i] = fmt.Sprintf("%d: %s", i, c.Prompts[i].Title)
	}

	c._sendDiscord(s, m, strings.Join(acts, "\n"))
	c._sendDiscord(s, m, "use prompts command: $ prompts!$act_key!")

	for i := 0; i < len(c.Prompts); i++ {
		q := fmt.Sprintf("%s!", c.Prompts[i].Title)
		if strings.Contains(m.Content, q) {
			command := c.Prompts[i].Replace("user", "assistant", "3").Command
			log.Infof("set act: %s", c.Prompts[i].Title, command)
			if res := c.Request(SYSTEM, command); res != "" {
				c._sendDiscord(s, m, fmt.Sprintf("success set actor: %s, say %s", c.Prompts[i].Title, res))
			} else {
				c._sendDiscord(s, m, fmt.Sprintf("fail set actor: %s, res: %s", c.Prompts[i].Title, res))
			}
			return
		}
	}
}

func (c *Client) _sendDiscord(s *discordgo.Session, m *discordgo.MessageCreate, q string) {
	l := int(math.Ceil(float64(len(q)) / float64(MAXLENGTH)))
	for i := 0; i < l; i++ {
		if len(q) > MAXLENGTH {
			s.ChannelMessageSend(m.ChannelID, q[:MAXLENGTH])
			q = q[MAXLENGTH:]
			time.Sleep(time.Second)
		} else {
			s.ChannelMessageSend(m.ChannelID, q)
			return
		}
	}
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
		Model: "gpt-3.5-turbo-16k",
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
		chats = []gogpt.ChatCompletionMessage{
			{
				Role:    uid,
				Content: q,
			},
		}
		return fmt.Sprintf("set paramater\n%s", res.Choices[0].Message.Content)
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
