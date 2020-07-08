package main

/* TODOs
0. Include all events
2. Make date/time more legible - page and events
3. Change to say "good morning/afternoon/evening"
4. Make clock live? Or we could just omit the clock
5. When does the token expire?
*/

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Event : a calendar event to display on page
type Event struct {
	Summary string
	Date    string
}

// Welcome : Name and time to display on page
type Welcome struct {
	TimeOfDay string
	Name      string
	Time      string
	Events    []Event
}

// Calendar token code: https://github.com/gsuitedevs/go-samples/blob/master/calendar/quickstart/quickstart.go

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
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
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	now := time.Now()
	t := now.Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(5).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next five of the user's events: %v", err)
	}

	var eventsStruct []Event
	initialLayout := "2006-01-02T15:04:05-07:00"
	finalLayout := "Mon, Jan _2 15:04"
	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			// Load event start time
			date := item.Start.DateTime
			if date == "" {
				continue
			}
			// Convert to time obj
			timeObj, err := time.Parse(initialLayout, date)
			if err != nil {
				fmt.Println(err)
			}

			// Re-format datetime
			date = timeObj.Format(finalLayout)
			// Append to Events slice
			eventsStruct = append(eventsStruct, Event{item.Summary, date})
		}
	}

	// Determine if it's the morning, afternoon, or evening
	// Morning: 2am -> noon. Afternoon: noon -> 6pm. Evening: 6pm -> 2am
	hour, _, _ := now.Clock()
	var timeOfDay string
	if (hour >= 2) && (hour < 12) {
		timeOfDay = "morning"
	} else if (hour >= 12) && (hour < 18) {
		timeOfDay = "afternoon"
	} else {
		timeOfDay = "evening"
	}

	// Fill in the Welcome struct
	welcome := Welcome{timeOfDay, "Mackenzie", now.Format(finalLayout), eventsStruct}

	// Load template, template.Must handles all excpetions/errors
	templates := template.Must(template.ParseFiles("templates/welcome-template.html"))

	// Setup http handler
	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// Handle function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// r.FormValue is the arguments in the URL
		if name := r.FormValue("name"); name != "" {
			welcome.Name = name
		}

		// Check for errors, templates is the template.must HTML file loaded
		if err := templates.ExecuteTemplate(w, "welcome-template.html", welcome); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Start server, listen to port 8080
	fmt.Println(http.ListenAndServe(":8080", nil))
}
