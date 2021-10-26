package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log"
	"mime/quotedprintable"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/DusanKasan/parsemail"
	"github.com/emersion/go-smtp"
	"github.com/skratchdot/open-golang/open"
	"github.com/toowoxx/go-lib-fs/fs"
)

//go:embed attachments.html
var attachmentsHtmlTemplate string

type AttachmentTemplateAttachment struct {
	Href string
	FileName string
}

type AttachmentTemplateOptions struct {
	Title string
	Attachments []AttachmentTemplateAttachment
}

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
		f, err := os.CreateTemp("", "devenv-mailserver-debug-mail-*.html")
		if err != nil {
			return err
		}

		if len(email.HTMLBody) > 0 {
			qpReader := quotedprintable.NewReader(
				bufio.NewReaderSize(
					strings.NewReader(email.HTMLBody),
					maxMessageBytes,
				),
			)
			_, err := io.Copy(f, qpReader)
			if err != nil {
				return err
			}
		} else {
			_, err := f.WriteString(fmt.Sprintf(
				"<!doctype html><html><head><meta charset=\"utf-8\"></head><body><pre>%s</pre></body></html>",
				email.TextBody))
			if err != nil {
				return err
			}
		}
		if err = f.Close(); err != nil {
			return err
		}
		log.Printf("Temporary file %s has been created", f.Name())
		log.Print("Starting default browser...")
		if err := open.Start(fmt.Sprintf("file://%s", f.Name())); err != nil {
			return err
		}

		if len(email.Attachments) > 0 {
			log.Print("Processing attachments...")
			tpl, err := template.New("attachments").Parse(attachmentsHtmlTemplate)
			if err != nil {
				return err
			}

			options := AttachmentTemplateOptions{}
			options.Title = email.Subject

			for _, attachment := range email.Attachments {
				log.Printf("Processing attachment %s", attachment.Filename)
				f, err := os.CreateTemp("", fmt.Sprintf("%s-*.%s",
						fs.RemoveExtension(attachment.Filename), path.Ext(attachment.Filename)))
				if err != nil {
					return err
				}

				_, err = io.Copy(f, attachment.Data)
				if err != nil {
					return err
				}

				options.Attachments = append(options.Attachments, AttachmentTemplateAttachment{
					Href:     f.Name(),
					FileName: attachment.Filename,
				})

				_ = f.Close()
			}

			log.Print("Creating attachment file")
			f, err := os.CreateTemp("", "devenv-mailserver-debug-attach-*.html")
			if err != nil {
				return err
			}
			if err := tpl.Execute(f, options); err != nil {
				return err
			}

			_ = f.Close()

			log.Print("Opening attachment page in browser...")
			if err := open.Start(fmt.Sprintf("file://%s", f.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

const maxMessageBytes = 262144 * 1024

// main starts a simple and permissive mail server
// that saves received mail to disk and opens it in the browser
func main() {
	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = "localhost:2028"
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = maxMessageBytes
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true
	s.EnableSMTPUTF8 = true

	log.Printf("Started mail server at %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Print(err)
	}
}
