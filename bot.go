// Telegram Bot API helper
//
// referenced: https://core.telegram.org/bots/api
//
// created on : 2015.10.06.
//
// by meinside@gmail.com

package telegrambot

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	ApiBaseUrl         = "https://api.telegram.org/bot"
	FileBaseUrl        = "https://api.telegram.org/file/bot"
	DefaultWebhookPort = 443

	WebhookPath = "/telegram/bot/webhook"
)

const (
	RedactedString = "<REDACTED>" // confidential info will be displayed as this
)

// Bot
type Bot struct {
	// Telegram bot API's token
	token       string
	tokenHashed string

	// Webhook related stuffs
	webhookHost    string
	webhookPort    int
	webhookUrl     string
	webhookHandler func(webhook Webhook, err error)

	// print verbose log messages or not
	Verbose bool
}

// Get a new bot API client with given token string.
func NewClient(token string) *Bot {
	return &Bot{
		token:       token,
		tokenHashed: fmt.Sprintf("%x", md5.Sum([]byte(token))),
	}
}

// Set webhook url and certificate for receiving incoming updates.
//
// https://core.telegram.org/bots/api#setwebhook
//
// port should be one of: 443, 80, 88, or 8443.
func (b *Bot) SetWebhook(host string, port int, certFilepath string) (result ApiResult) {
	b.webhookHost = host
	b.webhookPort = port
	b.webhookUrl = b.getWebhookUrl()

	file, err := os.Open(certFilepath)
	if err != nil {
		panic(err.Error())
	}

	params := map[string]interface{}{
		"url":         b.webhookUrl,
		"certificate": file,
	}

	b.verbose("setting webhook url to: %s", b.webhookUrl)

	return b.requestResult("setWebhook", params)
}

// Delete webhook.
//
// https://core.telegram.org/bots/api#setwebhook
//
// (Function GetUpdates will not work if webhook is set, so in that case you'll need to delete it)
func (b *Bot) DeleteWebhook() (result ApiResult) {
	b.webhookHost = ""
	b.webhookPort = 0
	b.webhookUrl = ""

	params := map[string]interface{}{
		"url": "",
	}

	b.verbose("deleting webhook url")

	return b.requestResult("setWebhook", params)
}

// Start Webhook server(and wait forever).
//
// Function SetWebhook(host, port, certFilepath) should be called priorly to setup host, port, and certification file.
//
// Certification file(.pem) and a private key is needed.
//
// (https://core.telegram.org/bots/self-signed)
//
// Incoming webhooks will be received through webhookHandler function.
func (b *Bot) StartWebhookServerAndWait(certFilepath string, keyFilepath string, webhookHandler func(webhook Webhook, err error)) {
	b.verbose("starting webhook server on: %s (port: %d) ...", b.getWebhookPath(), b.webhookPort)

	// routing
	b.webhookHandler = webhookHandler
	http.HandleFunc(b.getWebhookPath(), b.handleWebhook)

	// start server
	if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", b.webhookPort), certFilepath, keyFilepath, nil); err != nil {
		panic(err.Error())
	}
}

// Get webhook path generated with hash.
func (b *Bot) getWebhookPath() string {
	return fmt.Sprintf("%s/%s", WebhookPath, b.tokenHashed)
}

// Get full URL of webhook interface.
func (b *Bot) getWebhookUrl() string {
	return fmt.Sprintf("https://%s:%d%s", b.webhookHost, b.webhookPort, b.getWebhookPath())
}

// Remove confidential info from given string.
func (b *Bot) redact(str string) string {
	tokenRemoved := strings.Replace(str, b.token, RedactedString, -1)
	redacted := strings.Replace(tokenRemoved, b.tokenHashed, RedactedString, -1)
	return redacted
}

// Print formatted log message. (only when Bot.Verbose == true)
func (b *Bot) verbose(str string, args ...interface{}) {
	if b.Verbose {
		fmt.Printf("> %s\n", b.redact(fmt.Sprintf(str, args...)))
	}
}

// Print formatted error message.
func (b *Bot) error(str string, args ...interface{}) {
	fmt.Printf("* %s\n", b.redact(fmt.Sprintf(str, args...)))
}
