package cmd

/**
 * Bare bones sendmail drop-in replacement borrowed from MailHog
 */

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"os"
	"os/user"
	"sync"
	"time"

	"github.com/axllent/mailpit/utils/logger"
	flag "github.com/spf13/pflag"
)

// Run the Mailpit sendmail replacement.
func Run() {
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	username := "nobody"
	user, err := user.Current()
	if err == nil && user != nil && len(user.Username) > 0 {
		username = user.Username
	}

	fromAddr := username + "@" + host
	smtpAddr := "localhost:1025"
	var recip []string

	// defaults from envars if provided
	if len(os.Getenv("MP_SENDMAIL_SMTP_ADDR")) > 0 {
		smtpAddr = os.Getenv("MP_SENDMAIL_SMTP_ADDR")
	}
	if len(os.Getenv("MP_SENDMAIL_FROM")) > 0 {
		fromAddr = os.Getenv("MP_SENDMAIL_FROM")
	}

	var verbose bool

	// override defaults from cli flags
	flag.StringVar(&smtpAddr, "smtp-addr", smtpAddr, "SMTP server address")
	flag.StringVarP(&fromAddr, "from", "f", fromAddr, "SMTP sender")
	flag.BoolP("long-i", "i", true, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-t", "t", true, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolVarP(&verbose, "verbose", "v", false, "Verbose mode (sends debug output to stderr)")
	flag.Parse()

	// allow recipient to be passed as an argument
	recip = flag.Args()

	if verbose {
		fmt.Fprintln(os.Stderr, smtpAddr, fromAddr)
	}

	body, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading stdin")
		os.Exit(11)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("error parsing message body: %s", err))
		os.Exit(11)
	}

	if len(recip) == 0 {
		// We only need to parse the message to get a recipient if none where
		// provided on the command line.
		recip = append(recip, msg.Header.Get("To"))
	}

	err = smtp.SendMail(smtpAddr, nil, fromAddr, recip, body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error sending mail")
		logger.Log().Fatal(err)
	}
}

func Perf() {
	start := time.Now()

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	username := "nobody"
	user, err := user.Current()
	if err == nil && user != nil && len(user.Username) > 0 {
		username = user.Username
	}

	fromAddr := username + "@" + host
	smtpAddr := "localhost:1025"
	var recip []string

	// defaults from envars if provided
	if len(os.Getenv("MP_SENDMAIL_SMTP_ADDR")) > 0 {
		smtpAddr = os.Getenv("MP_SENDMAIL_SMTP_ADDR")
	}
	if len(os.Getenv("MP_SENDMAIL_FROM")) > 0 {
		fromAddr = os.Getenv("MP_SENDMAIL_FROM")
	}

	var verbose bool

	emailSize := 128
	concurrencyNum := 8
	perConcurrencyCnt := 128
	reuseConnection := false

	// override defaults from cli flags
	flag.StringVar(&smtpAddr, "smtp-addr", smtpAddr, "SMTP server address")
	flag.StringVarP(&fromAddr, "from", "f", "", "SMTP sender")
	flag.IntVar(&emailSize, "emailsize", 128, "The mailsize")
	flag.IntVar(&concurrencyNum, "concurrencyNum", 8, "The concurrencyNum")
	flag.IntVar(&perConcurrencyCnt, "perConcurrencyCnt", 128, "The perConcurrencyCnt")
	flag.BoolVar(&reuseConnection, "reuseConnection", false, "If to reuse the mail dial connection")
	flag.BoolP("long-i", "i", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-t", "t", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolVarP(&verbose, "verbose", "v", false, "Verbose mode (sends debug output to stderr)")
	flag.Parse()

	// allow recipient to be passed as an argument
	recip = flag.Args()

	if verbose {
		fmt.Fprintln(os.Stderr, smtpAddr, fromAddr, emailSize, concurrencyNum, perConcurrencyCnt)
	}

	pre := "From: App <app@mailhog.local>\nTo: Test <test@mailhog.local>\nSubject: Test message\n\n"
	preLen := len(pre)
	body := make([]byte, emailSize + preLen)
	copy(body, pre)
	for i := 0; i < emailSize; i += 1 {
		body[i + preLen] = 'A'
	}

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("error parsing message body: %s", err))
		os.Exit(11)
	}

	if len(recip) == 0 {
		// We only need to parse the message to get a recipient if none where
		// provided on the command line.
		recip = append(recip, msg.Header.Get("To"))
	}
	var wg sync.WaitGroup
	for i:= 0; i < concurrencyNum; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if reuseConnection {
				SendMailHoldCon(smtpAddr, fromAddr, perConcurrencyCnt, recip, body)
			} else {
				SendMailNoHoldCon(smtpAddr, fromAddr, perConcurrencyCnt, recip, body)
			}
		}()
	}
	wg.Wait()


	currentTime := time.Now()
	diff := currentTime.Sub(start)
	fmt.Printf("QPS: %d", (int)(float64(concurrencyNum) * float64(perConcurrencyCnt) / diff.Seconds()))
}

func SendMailNoHoldCon(smtpAddr string, fromAddr string, perConcurrencyCnt int, recip []string, body []byte) {
	for j := 0; j < perConcurrencyCnt; j += 1 {
		err := smtp.SendMail(smtpAddr, nil, fromAddr, recip, body)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error sending mail")
			logger.Log().Fatal(err)
		}
	}
 }

func SendMailHoldCon(smtpAddr string, fromAddr string, perConcurrencyCnt int, recip []string, msg []byte) {
	c, err := smtp.Dial(smtpAddr)
	if err != nil {
		if err = c.Hello(fromAddr); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("error connecting smtp server: %s", err))
			return
		}
	}

	defer c.Quit()
	for j := 0; j < perConcurrencyCnt; j += 1 {
		err = SendMail(c, fromAddr, recip, msg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error sending mail")
			logger.Log().Fatal(err)
		}
	}
}

func SendMail(c *smtp.Client, fromAddr string, recip []string, msg []byte) error {
	var err error
	if err = c.Mail(fromAddr); err != nil {
		fmt.Fprintln(os.Stderr, "error mail call")
		return err
	}
	for _, addr := range recip {
		if err = c.Rcpt(addr); err != nil {
			fmt.Fprintln(os.Stderr, "error rcpt call")
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error data call")
		return err
	}
	defer w.Close()
	_, err = w.Write(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error write call")
		return err
	}
	return nil
}
