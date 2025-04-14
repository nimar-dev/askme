package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
)

type User struct {
	ID       int
	Phone    string
	Language string
	Concern  string
	Stage    int // 0: Greeted, 1: Language, 2: Concern
}

type Therapist struct {
	ID        int
	Name      string
	Language  string
	Specialty string
	Slots     []string
	WhatsApp  string
}

var users = make(map[string]*User) // phone -> user

var therapists = []Therapist{
	{ID: 1, Name: "Dr. Asha", Language: "Hindi", Specialty: "Anxiety", Slots: []string{"6PM", "9PM"}, WhatsApp: "918888888888"},
	{ID: 2, Name: "Dr. Raj", Language: "English", Specialty: "Relationships", Slots: []string{"5PM"}, WhatsApp: "919999999999"},
}

type WhatsAppMessage struct {
	From string `json:"from"`
	Text string `json:"text"`
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	var msg WhatsAppMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, exists := users[msg.From]
	if !exists {
		user = &User{Phone: msg.From, Stage: 0}
		users[msg.From] = user
	}
	fmt.Printf("Received message from %s: %s\n", msg.From, msg.Text)
	switch user.Stage {
	case 0:
		fmt.Println("New user detected, sending greeting.")
		sendWhatsApp(msg.From, "Hi! Welcome to MindMitra 💚\nWhat language do you prefer?\n1. Hindi\n2. English")
		user.Stage = 1
	case 1:
		fmt.Println("User selected language.")
		if msg.Text == "1" {
			user.Language = "Hindi"
		} else {
			user.Language = "English"
		}
		sendWhatsApp(msg.From, "What would you like help with?\n1. Stress\n2. Relationship\n3. Anxiety")
		user.Stage = 2
	case 2:
		fmt.Println("User selected concern.")
		if msg.Text == "1" {
			user.Concern = "Stress"
		} else if msg.Text == "2" {
			user.Concern = "Relationship"
		} else {
			user.Concern = "Anxiety"
		}
		matchAndSendTherapist(msg.From, user)
		user.Stage = 3
	default:
		fmt.Println("User already matched with a therapist.")
		sendWhatsApp(msg.From, "Thank you! We're setting up your session. You'll get details shortly.")
	}
}

func matchAndSendTherapist(phone string, user *User) {
	for _, t := range therapists {
		if t.Language == user.Language && t.Specialty == user.Concern {
			msg := fmt.Sprintf("We found a therapist for you: %s (%s)\nAvailable today at %s.\nSession Fee: ₹199. Reply YES to confirm.", t.Name, t.Specialty, t.Slots[0])
			sendWhatsApp(phone, msg)
			return
		}
	}
	sendWhatsApp(phone, "Sorry, we couldn't find a therapist right now. We'll reach out when one is available.")
}

func sendWhatsApp(to string, message string) {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	from := "whatsapp:+14155238886" // Twilio Sandbox WhatsApp number
	to = "whatsapp:" + to           // Format the recipient's number

	fmt.Printf("HiiiiiiSending message to %s: %s\n", to, message)
	url := "https://api.twilio.com/2010-04-01/Accounts/" + accountSID + "/Messages.json"

	data := neturl.Values{}
	data.Set("From", from)
	data.Set("To", to)
	data.Set("Body", message)

	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return
	}
	req.SetBasicAuth(accountSID, authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending message: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("Message sent successfully!")
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to send message: %s\n", body)
	}
}

func main() {
	http.HandleFunc("/webhook", handleWebhook)
	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
