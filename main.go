package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type StartDate struct {
	StartDate string `json:"start_date"`
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables.")
	}
}

func getDayCount(filename string) int {
	// If file doesn't exist, set today as the start date
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		today := time.Now().Format("2006-01-02")
		startData := StartDate{StartDate: today}
		data, _ := json.Marshal(startData)
		_ = os.WriteFile(filename, data, 0644)
		fmt.Println("First time running! Setting today as Day 1.")
		return 1
	}

	// Read start date from file
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read start date file: %v", err)
	}

	var start StartDate
	err = json.Unmarshal(data, &start)
	if err != nil {
		log.Fatalf("Failed to parse start date file: %v", err)
	}

	// Parse using local timezone
	startDate, err := time.ParseInLocation("2006-01-02", start.StartDate, time.Local)
	if err != nil {
		log.Fatalf("Failed to parse start date: %v", err)
	}

	// Calculate elapsed time in days as a float
	duration := time.Since(startDate).Hours() / 24.0

	// Set dayCount to the integer part + 1
	// As per current hosting in GMT I may need to get rid of the plus one
	// or maybe rework how time is found entirely instead of OS based
	dayCount := int(duration) + 1

	fmt.Println("Start Date:", startDate.Format("2006-01-02"))
	fmt.Printf("Elapsed Days: %.6f\n", duration)
	fmt.Println("Calculated Day Count:", dayCount)
	return dayCount
}

func getTodaysMessage(filename string, day int) (int, string, bool) {
	fmt.Println("Reading messages.json file...")

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var messages struct {
		Messages []struct {
			Day  int    `json:"day"`
			Text string `json:"text"`
		} `json:"messages"`
	}

	err = json.Unmarshal(data, &messages)
	if err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	for _, msg := range messages.Messages {
		if msg.Day == day {
			fmt.Println("Found message for Day", day)
			return day, msg.Text, true
		}
	}

	fmt.Println("No message found for Day", day)
	return day, "", false
}

func sendEmail(day int, message string) error {
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587" // default to 587 if not set
	}
	senderEmail := os.Getenv("SENDER_EMAIL")
	senderPass := os.Getenv("SENDER_PASS")
	receiverEmail := os.Getenv("RECEIVER_EMAIL")

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer,
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}

	fmt.Println("Using plain connection for STARTTLS...")
	conn, err := dialer.Dial("tcp", smtpServer+":"+smtpPort)
	if err != nil {
		fmt.Println("Error dialing SMTP server:", err)
		return err
	}
	// Set an initial deadline to avoid hanging indefinitely
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer conn.Close()
	fmt.Println("Dialed SMTP server successfully.")

	client, err := smtp.NewClient(conn, smtpServer)
	if err != nil {
		fmt.Println("Error creating SMTP client:", err)
		return err
	}
	defer client.Quit()
	fmt.Println("SMTP client created.")

	if ok, _ := client.Extension("STARTTLS"); ok {
		// Set a longer deadline for the TLS handshake (60 seconds)
		conn.SetDeadline(time.Now().Add(60 * time.Second))
		fmt.Println("Starting TLS via STARTTLS...")
		if err = client.StartTLS(tlsConfig); err != nil {
			fmt.Println("Error starting TLS:", err)
			return err
		}
		// Clear the deadline after handshake completes
		conn.SetDeadline(time.Time{})
		fmt.Println("TLS started.")
	} else {
		fmt.Println("STARTTLS not supported, proceeding without TLS upgrade.")
	}

	auth := smtp.PlainAuth("", senderEmail, senderPass, smtpServer)
	fmt.Println("Authenticating...")
	if err = client.Auth(auth); err != nil {
		fmt.Println("Error during authentication:", err)
		return err
	}
	fmt.Println("Authenticated.")

	fmt.Println("Setting MAIL from address...")
	if err = client.Mail(senderEmail); err != nil {
		fmt.Println("Error setting sender:", err)
		return err
	}
	fmt.Println("Setting RCPT recipient address...")
	if err = client.Rcpt(receiverEmail); err != nil {
		fmt.Println("Error setting recipient:", err)
		return err
	}

	wc, err := client.Data()
	if err != nil {
		fmt.Println("Error obtaining Data writer:", err)
		return err
	}
	defer wc.Close()

	subject := fmt.Sprintf("Mailentine Day #%d 💌", day)
	body := message
	msgData := []byte("To: " + receiverEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	fmt.Println("Writing message data...")
	if _, err = wc.Write(msgData); err != nil {
		fmt.Println("Error writing message:", err)
		return err
	}

	fmt.Println("Email sent successfully!")
	return nil
}

func emailHandler(w http.ResponseWriter, r *http.Request) {
	day := getDayCount("start_date.json")
	day, message, hasMessage := getTodaysMessage("messages.json", day)

	if !hasMessage {
		fmt.Println("No message for today. Skipping email.")
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "No email sent for today.")
		return
	}

	err := sendEmail(day, message)
	if err != nil {
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Email sent for Day %d", day)
}

func main() {
	loadEnv()

	// Health check handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Service is up")
	})

	http.HandleFunc("/send-email", emailHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server starting on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
