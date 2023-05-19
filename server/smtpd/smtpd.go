// Package smtpd is the SMTP daemon
package smtpd

import (
	"bytes"
	"net"
	"net/mail"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/storage"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/theidexisted/smtpd"
	"net/http"
)

var (
	MailReceived = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "received_mails_total",
		Help: "The total number of received mails",
		},
		[]string{"source_ip"},
	)
	AccumulatedSession = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "accumulate_session_cnt",
		Help: "The total number of history session",
		},
		[]string{"source_ip"},
	)
	SessionNum = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "realtime_session",
		Help: "The realtime mail client sessions",
		},
		[]string{"source_ip"},
	)
	MailReceiveLatencyHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "received_mails_latency_hist",
		Help:    "The latency(in us) of received mails(hello to data transfer done)",
		Buckets: prometheus.ExponentialBuckets(200, 1.8, 15),
	},
	[]string{"source_ip"},
	)
	SimulateReceiveLatencyUS int64 = 0
)

func ExtractIP(addr net.Addr) string {
	return strings.Split(addr.String(), ":")[0]
}

func mailSessionOpHandler(origin net.Addr, isNewConnection bool) {
	if isNewConnection {
		SessionNum.WithLabelValues(ExtractIP(origin)).Inc()
		AccumulatedSession.WithLabelValues(ExtractIP(origin)).Inc()
	} else {
		SessionNum.WithLabelValues(ExtractIP(origin)).Dec()
	}
}

func mailReceiveLatency(origin net.Addr, elapsed time.Duration) {
	MailReceiveLatencyHist.WithLabelValues(ExtractIP(origin)).Observe(float64(elapsed.Microseconds()))
}

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		logger.Log().Errorf("error parsing message: %s", err.Error())
		return err
	}

	if _, err := storage.Store(data); err != nil {
		// Value with size 4800709 exceeded 1048576 limit
		re := regexp.MustCompile(`(Value with size \d+ exceeded \d+ limit)`)
		tooLarge := re.FindStringSubmatch(err.Error())
		if len(tooLarge) > 0 {
			logger.Log().Errorf("[db] error storing message: %s", tooLarge[0])
		} else {
			logger.Log().Errorf("[db] error storing message")
			logger.Log().Errorf(err.Error())
		}
		return err
	}
	lat := time.Microsecond * time.Duration(atomic.LoadInt64(&SimulateReceiveLatencyUS))
	logger.Log().Debugf("[smtp] Simulate server latency with sleep :%s", lat)
	time.Sleep(lat)
	MailReceived.WithLabelValues(ExtractIP(origin)).Inc()

	subject := msg.Header.Get("Subject")
	logger.Log().Debugf("[smtp] received (%s) from:%s to:%s subject:%q", cleanIP(origin), from, to[0], subject)
	return nil
}

func authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	allow := config.SMTPAuth.Match(string(username), string(password))
	if allow {
		logger.Log().Debugf("[smtp] allow %s login:%q from:%s", mechanism, string(username), cleanIP(remoteAddr))
	} else {
		logger.Log().Warnf("[smtp] deny %s login:%q from:%s", mechanism, string(username), cleanIP(remoteAddr))
	}
	return allow, nil
}

// Allow any username and password
func authHandlerAny(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error) {
	logger.Log().Debugf("[smtp] allow %s login %q from %s", mechanism, string(username), cleanIP(remoteAddr))
	return true, nil
}

// Listen starts the SMTPD server
func Listen() error {
	globalReg := prometheus.NewRegistry()
	globalReg.MustRegister(MailReceived)
	globalReg.MustRegister(AccumulatedSession)
	globalReg.MustRegister(SessionNum)
	globalReg.MustRegister(MailReceiveLatencyHist)
	go func() {
		logger.Log().Infof("[http] starting metrics server on http://localhost:2112/metrics")
		http.Handle("/metrics", promhttp.HandlerFor(globalReg, promhttp.HandlerOpts{}))
		http.ListenAndServe(":2112", nil)
	}()

	if config.SMTPAuthAllowInsecure {
		if config.SMTPAuthFile != "" {
			logger.Log().Infof("[smtp] enabling login auth via %s (insecure)", config.SMTPAuthFile)
		} else if config.SMTPAuthAcceptAny {
			logger.Log().Info("[smtp] enabling all auth (insecure)")
		}
	} else {
		if config.SMTPAuthFile != "" {
			logger.Log().Infof("[smtp] enabling login auth via %s (TLS)", config.SMTPAuthFile)
		} else if config.SMTPAuthAcceptAny {
			logger.Log().Info("[smtp] enabling any auth (TLS)")
		}
	}

	logger.Log().Infof("[smtp] starting on %s", config.SMTPListen)

	return listenAndServe(config.SMTPListen, mailHandler, authHandler)
}

func listenAndServe(addr string, handler smtpd.Handler, authHandler smtpd.AuthHandler) error {
	srv := &smtpd.Server{
		Addr:                  addr,
		Handler:               handler,
		ReceiveLatencyHandler: mailReceiveLatency,
		SessionOpHandler:      mailSessionOpHandler,
		Appname:               "Mailpit",
		Hostname:              "",
		AuthHandler:           nil,
		AuthRequired:          false,
	}

	if config.SMTPAuthAllowInsecure {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
	}

	if config.SMTPAuthFile != "" {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
		srv.AuthHandler = authHandler
		srv.AuthRequired = true
	} else if config.SMTPAuthAcceptAny {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
		srv.AuthHandler = authHandlerAny
	}

	if config.SMTPTLSCert != "" {
		if err := srv.ConfigureTLS(config.SMTPTLSCert, config.SMTPTLSKey); err != nil {
			return err
		}
	}

	return srv.ListenAndServe()
}

func cleanIP(i net.Addr) string {
	parts := strings.Split(i.String(), ":")

	return parts[0]
}
