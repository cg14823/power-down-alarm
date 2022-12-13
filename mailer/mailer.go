package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"gopkg.in/gomail.v2"
)

const timeFormat = "02/01/2006 15:04"

const outageEmailBodyTemplate = `
There has been a power cut lasting {{.Duration}} at 59 Great Ancoats Street.
Please remember to update the tracking spreadsheet with the following details:

Start: {{.Start}}
End:   {{.End}}
`

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokenFilePath string) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenFilePath)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}

		if err = saveToken(tokenFilePath, tok); err != nil {
			return nil, err
		}
	}

	return config.Client(context.Background(), tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}

	return tok, nil
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

type Mailer struct {
	to           string
	from         string
	gmailService *gmail.Service
	logger       *zap.Logger
}

func NewMailer(to, from, credentialsFile, tokenFile string, logger *zap.Logger) (*Mailer, error) {
	ctx := context.Background()
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, err
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailComposeScope)
	if err != nil {
		return nil, err
	}

	client, err := getClient(config, tokenFile)
	if err != nil {
		return nil, err
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &Mailer{
		to:           to,
		from:         from,
		gmailService: srv,
		logger:       logger,
	}, nil
}

func (m *Mailer) sendEmail(start, end time.Time) error {
	buff := bytes.NewBuffer([]byte{})
	err := template.Must(template.New("email").Parse(outageEmailBodyTemplate)).Execute(buff, struct {
		Start    string
		End      string
		Duration time.Duration
	}{
		Start:    start.Format(timeFormat),
		End:      end.Format(timeFormat),
		Duration: end.Sub(start).Round(time.Minute),
	})
	if err != nil {
		return fmt.Errorf("could not create email body: %w", err)
	}

	var message = gomail.NewMessage()
	message.SetHeader("From", m.from)
	message.SetHeader("To", m.to)
	message.SetHeader("Subject", "Power outage")
	message.SetBody("text/plain", buff.String())

	buffer := new(bytes.Buffer)
	if _, err = message.WriteTo(buffer); err != nil {
		return fmt.Errorf("could not create email: %w", err)
	}

	var msg gmail.Message
	msg.Raw = base64.URLEncoding.EncodeToString(buffer.Bytes())

	_, err = m.gmailService.Users.Messages.Send(m.from, &msg).Do()
	if err != nil {
		return fmt.Errorf("could not send email: %w", err)
	}

	return nil
}

func (m *Mailer) AysncOutageNotification(start, end time.Time) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		var err error
		for i := 0; i < 5; i++ {
			err = m.sendEmail(start, end)
			if err == nil {
				m.logger.Info("Email sent")
				return
			}

			m.logger.Sugar().Warnw("could not send email", "err", err.Error())
			time.Sleep(time.Duration(10*(1+i)) * time.Second)
		}

		if err != nil {
			m.logger.Sugar().Errorw("could not send email", "err", err.Error())
		}
	}()

	return done
}
