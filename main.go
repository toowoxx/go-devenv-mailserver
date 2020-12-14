package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/quotedprintable"
	"strings"
	"time"

	"github.com/DusanKasan/parsemail"
	"github.com/emersion/go-smtp"
	"github.com/skratchdot/open-golang/open"
)

// The Backend implements SMTP server methods.
type Backend struct{}

// Login handles a login command with username and password.
func (bkd *Backend) Login(_ *smtp.ConnectionState, _, _ string) (smtp.Session, error) {
	return &Session{}, nil
}

// AnonymousLogin handles anonymous logins. They are always accepted.
func (bkd *Backend) AnonymousLogin(_ *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{}, nil
}

// A Session is returned after successful login.
type Session struct{}

// Mail is called on mail from:
func (s *Session) Mail(from string, _ smtp.MailOptions) error {
	log.Printf("mail from: %s", from)
	return nil
}

// Rcpt is called on rcpt to:
func (s *Session) Rcpt(to string) error {
	log.Printf("rcpt to: %s", to)
	return nil
}

// Data is called when the client is ready to transmit the data
func (s *Session) Data(r io.Reader) error {
	if email, err := parsemail.Parse(r); err != nil {
		return err
	} else {
		log.Print("Received new email: ",
			map[string]interface{}{
				"Subject": email.Subject,
				"Date": email.Date,
			},
		)
		log.Print("data received, writing to file")
		f, err := ioutil.TempFile("", "devenv-mailserver-debug-mail-*.html")
		if err != nil {
			return err
		}
		qpReader := quotedprintable.NewReader(strings.NewReader(email.HTMLBody))
		qpStrBytes, err := ioutil.ReadAll(qpReader)
		if err != nil {
			return err
		}
		if _, err = f.Write(qpStrBytes); err != nil {
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
		log.Printf("Temporary file %s has been created", f.Name())
		log.Print("Starting default browser...")
		return open.Start(fmt.Sprintf("file://%s", f.Name()))
	}
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

// main starts a simple and permissive mail server
// that saves received mail to disk and opens it in the browser
func main() {
	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = "localhost:2028"
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 4096 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true
	s.EnableSMTPUTF8 = true

	log.Printf("Started mail server at %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Print(err)
	}
}
