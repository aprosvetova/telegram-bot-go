// Package telegrambot / Telegram Bot API helper
//
// https://core.telegram.org/bots/api
//
// Created on : 2015.10.06, meinside@gmail.com
package telegrambot

import (
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	apiBaseURL  = "https://api.telegram.org/bot"
	fileBaseURL = "https://api.telegram.org/file/bot"

	webhookPath = "/telegram/bot/webhook"
)

const (
	redactedString = "<REDACTED>" // confidential info will be displayed as this
)

// Bot struct
type Bot struct {
	token       string // Telegram bot API's token
	tokenHashed string // hashed token

	webhookHost string // webhook hostname
	webhookPort int    // webhook port number
	webhookURL  string // webhook url

	httpClient *http.Client // http client

	updateHandler func(b *Bot, update Update, err error) // update(webhook) handler function

	Verbose bool // print verbose log messages or not
}

// NewClient gets a new bot API client with given token string.
func NewClient(token string) *Bot {
	return &Bot{
		token:       token,
		tokenHashed: fmt.Sprintf("%x", md5.Sum([]byte(token))),

		httpClient: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 300 * time.Second,
				}).Dial,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

// GenCertAndKey generates a certificate and a private key file with given domain.
// (`OpenSSL` is needed.)
func GenCertAndKey(domain string, outCertFilepath string, outKeyFilepath string, expiresInDays int) error {
	numBits := 2048
	country := "US"
	state := "New York"
	local := "Brooklyn"
	org := "Example Company"

	if _, err := exec.Command("openssl", "req", "-newkey", fmt.Sprintf("rsa:%d", numBits), "-sha256", "-nodes", "-keyout", outKeyFilepath, "-x509", "-days", strconv.Itoa(expiresInDays), "-out", outCertFilepath, "-subj", fmt.Sprintf("/C=%s/ST=%s/L=%s/O=%s/CN=%s", country, state, local, org, domain)).Output(); err != nil {
		return err
	}

	return nil
}

// StartWebhookServerAndWait starts a webhook server(and waits forever).
// Function SetWebhook(host, port, certFilepath) should be called priorly to setup host, port, and certification file.
// Certification file(.pem) and a private key is needed.
// Incoming webhooks will be received through webhookHandler function.
//
// https://core.telegram.org/bots/self-signed
func (b *Bot) StartWebhookServerAndWait(certFilepath string, keyFilepath string, webhookHandler func(b *Bot, webhook Update, err error)) {
	b.verbose("starting webhook server on: %s (port: %d) ...", b.getWebhookPath(), b.webhookPort)

	// set update handler
	b.updateHandler = webhookHandler

	// routing
	mux := http.NewServeMux()
	mux.HandleFunc(b.getWebhookPath(), b.handleWebhook)

	// start server
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", b.webhookPort),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServeTLS(certFilepath, keyFilepath); err != nil {
		panic(err.Error())
	}
}

// StartMonitoringUpdates retrieves updates from API server constantly.
// If webhook is registered, it may not work properly. So make sure webhook is deleted, or not registered.
func (b *Bot) StartMonitoringUpdates(updateOffset int, interval int, updateHandler func(b *Bot, update Update, err error)) {
	b.verbose("starting monitoring updates (interval seconds: %d) ...", interval)

	options := OptionsGetUpdates{}.SetOffset(updateOffset)

	// set update handler
	b.updateHandler = updateHandler

	var updates APIResponseUpdates
	for {
		if updates = b.GetUpdates(options); updates.Ok {
			for _, update := range updates.Result {
				// update offset (max + 1)
				if options["offset"].(int) <= update.UpdateID {
					options["offset"] = update.UpdateID + 1
				}

				go b.updateHandler(b, update, nil)
			}
		} else {
			go b.updateHandler(b, Update{}, fmt.Errorf("error while retrieving updates - %s", *updates.Description))
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

// Get webhook path generated with hash.
func (b *Bot) getWebhookPath() string {
	return fmt.Sprintf("%s/%s", webhookPath, b.tokenHashed)
}

// Get full URL of webhook interface.
func (b *Bot) getWebhookURL() string {
	return fmt.Sprintf("https://%s:%d%s", b.webhookHost, b.webhookPort, b.getWebhookPath())
}

// Remove confidential info from given string.
func (b *Bot) redact(str string) string {
	tokenRemoved := strings.Replace(str, b.token, redactedString, -1)
	redacted := strings.Replace(tokenRemoved, b.tokenHashed, redactedString, -1)
	return redacted
}

// Print formatted log message. (only when Bot.Verbose == true)
func (b *Bot) verbose(str string, args ...interface{}) {
	if b.Verbose {
		log.Printf("> %s\n", b.redact(fmt.Sprintf(str, args...)))
	}
}

// Print formatted error message.
func (b *Bot) error(str string, args ...interface{}) {
	log.Printf("* %s\n", b.redact(fmt.Sprintf(str, args...)))
}
