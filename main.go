// This is a Kris Kringle service. It takes a list of emails, assigns each email a partner, and sends them the
// email of their partner using the email package (SMTP).

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	log "github.com/sirupsen/logrus"
	mail "github.com/xhit/go-simple-mail/v2"
)

const htmlBody = `<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<title>Your Kris Kringle partner</title>
	</head>
	<body>
		<h1>Hi there!</h1>
		<p>
			The email of your Kris Kringle partner is: <a href="mailto:%s">%s</a>.
		</p>

		<p>
			Remember, your Kris Kringle gift should be below 10 dollars and ready by the 20th of December.
		</p>

		<p>
			We hope you enjoyed the Kris Kringle service and look forward to doing business with you next year.
		</p>

		<p>
			Sincerely,<br/>
			Santa Claus
		</p>

		<p>
			<code>This message is valid as of %s.</code>
		</p>
	</body>
</html>`

var smtpClient *mail.SMTPClient

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
	})

	log.SetLevel(log.DebugLevel)
	rand.Seed(time.Now().UnixNano())

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	server := mail.NewSMTPClient()

	// SMTP Server
	server.Host = "smtp.gmail.com"
	server.Port = 587
	server.Username = os.Getenv("EMAIL_USER")
	server.Password = os.Getenv("EMAIL_PASSWORD")
	server.Encryption = mail.EncryptionSTARTTLS

	// Since v2.3.0 you can specified authentication type:
	// - PLAIN (default)
	// - LOGIN
	// - CRAM-MD5
	// - None
	// server.Authentication = mail.AuthPlain

	// Variable to keep alive connection
	server.KeepAlive = true

	// Timeout for connect to SMTP Server
	server.ConnectTimeout = 10 * time.Second

	// Timeout for send the data and wait respond
	server.SendTimeout = 10 * time.Second

	// Set TLSConfig to provide custom TLS configuration. For example,
	// to skip TLS verification (useful for testing):
	server.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// SMTP client
	smtpClient, err = server.Connect()

	log.Info("Connected to SMTP server")

	if err != nil {
		log.Fatal(err)
	}

}

func sendKringleEmail(recipient string, partner string) {

	// New email simple html with inline and CC
	email := mail.NewMSG()
	email.SetFrom("shaunak's kris kringle server (might be buggy) <kringle@srg.id.au>").
		AddTo(recipient).
		// AddBcc("gadkari.shaunak+kingle_reflect@gmail.com").
		SetSubject("Your Kris Kringle for this year")

	email.SetBody(mail.TextHTML, fmt.Sprintf(htmlBody, partner, partner, time.Now().Format("2006-01-02 15:04:05")))

	// also you can add body from []byte with SetBodyData, example:
	// email.SetBodyData(mail.TextHTML, []byte(htmlBody))
	// or alternative part
	// email.AddAlternativeData(mail.TextHTML, []byte(htmlBody))

	// add inline
	// email.Attach(&mail.File{FilePath: "/path/to/image.png", Name: "Gopher.png", Inline: true})

	// always check error after send
	if email.Error != nil {
		log.WithFields(log.Fields{
			"recipient": recipient,
			"partner":   partner,
		}).Fatal(email.Error)
	}

	// Call Send and pass the client
	err := email.Send(smtpClient)
	if err != nil {
		log.WithFields(log.Fields{
			"recipient": recipient,
			"partner":   partner,
		}).Fatal(err)
	} else {
		log.Println("Email Sent")
	}

}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func main() {

	type KringleRequest struct {
		Emails []string
	}

	// Define a HTTP server that gets a POST request with a JSON body, containing a list of emails.

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Access-Control-Allow-Origin", "*")

		var k KringleRequest

		err := json.NewDecoder(r.Body).Decode(&k)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Fatal(err)
			return
		}

		log.WithField("emails", k.Emails).Info("Received emails")

    if len(k.Emails) > 10 {
      w.WriteHeader(http.StatusBadRequest)
      w.Write([]byte("Up to 10 emails at once only."))
    }

		// Create a channel to communicate between the goroutines.
		// statusChannel := make(chan string)

		// k.Emails is an array with n emails.

		// First, randomly shuffle the emails.
		rand.Shuffle(len(k.Emails), func(i, j int) { k.Emails[i], k.Emails[j] = k.Emails[j], k.Emails[i] })

		// Define a slice to keep track of who has already been a giver.
		givers := []string{}

		// Loop through each email in the emails slice.
		for index, giver := range k.Emails {

			// If the giver has already been a giver, then break out of the loop.
			if contains(givers, giver) {
				log.Info("Finished.")
				break
			}

			// Append the giver to the givers slice.
			givers = append(givers, giver)

			// The next email in the slice is the recipient.
			recipient := k.Emails[(index+1)%len(k.Emails)]

			log.WithFields(log.Fields{
				"giver":     giver,
				"recipient": recipient,
			}).Info("Starting to send email")

			// Send the email (concurrently)
			sendKringleEmail(giver, recipient)

		}

		// // // Wait for all emails to be sent
		// for i := 0; i < len(k.Emails); i++ {
		// 	log.Info(<-statusChannel)
		// }

		log.Info("Done")

		// Return a 200 OK status
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}
