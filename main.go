package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

type User struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Note struct {
	ID     uint32 `json:"id"`
	Note   string `json:"note"`
	UserID uint32 `json:"userID"`
}

type Session struct {
	ID       string `json:"sid"`
	UserID   uint32 `json:"userID"`
	LoggedIn bool   `json:"-"`
}

var (
	users     []User
	notes     []Note
	sessions  []Session
	userMutex sync.Mutex
	noteMutex sync.Mutex
)

func main() {
	// Create a new Gorilla Mux router
	router := mux.NewRouter()

	// Define the routes
	router.HandleFunc("/signup", signup).Methods("POST")
	router.HandleFunc("/login", login).Methods("POST")
	router.HandleFunc("/notes", listNotes).Methods("GET")
	router.HandleFunc("/notes", createNote).Methods("POST")
	router.HandleFunc("/notes", deleteNote).Methods("DELETE")

	// Start the server
	log.Fatal(http.ListenAndServe(":3000", router))
}

func signup(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a User struct
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)

	if newUser.Name == "" && newUser.Email == "" && newUser.Password == "" {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the user
	newUser.ID = uint32(len(users) + 1)

	// Add the user to the users slice
	userMutex.Lock()
	users = append(users, newUser)
	userMutex.Unlock()

	// Set the response header and send the response as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(http.StatusOK)
}

func login(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a LoginRequest struct
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&loginReq)
	if loginReq.Email == "" && loginReq.Password == "" {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find the user with the matching email and password
	userMutex.Lock()
	var loggedInUser User
	for _, user := range users {
		if user.Email == loginReq.Email && user.Password == loginReq.Password {
			loggedInUser = user
			break
		}
	}
	userMutex.Unlock()

	// If no user found, return unauthorized
	if loggedInUser.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate a unique session ID
	sessionID := strconv.Itoa(len(sessions) + 1)

	// Add the session to the sessions slice
	session := Session{
		ID:       sessionID,
		UserID:   loggedInUser.ID,
		LoggedIn: true,
	}
	sessions = append(sessions, session)

	// Create the response
	response := struct {
		SessionID string `json:"sid"`
	}{
		SessionID: sessionID,
	}

	// Set the response header and send the response as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listNotes(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a SessionRequest struct
	var sessionReq struct {
		SessionID string `json:"sid"`
	}
	err := json.NewDecoder(r.Body).Decode(&sessionReq)
	if sessionReq.SessionID == "" {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find the session with the matching session ID
	noteMutex.Lock()
	var userID uint32
	sessionFound := false
	for _, session := range sessions {
		if session.ID == sessionReq.SessionID {
			sessionFound = true
			userID = session.UserID
			break
		}
	}
	noteMutex.Unlock()

	// If session not found, return unauthorized
	if !sessionFound {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get all notes for the user with matching user ID
	userNotes := make([]Note, 0)
	for _, note := range notes {
		if note.UserID == userID {
			userNotes = append(userNotes, note)
		}
	}

	// Create the response
	response := struct {
		Notes []Note `json:"notes"`
	}{
		Notes: userNotes,
	}

	// Set the response header and send the response as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a NoteRequest struct
	var noteReq struct {
		SessionID string `json:"sid"`
		Note      string `json:"note"`
	}

	err := json.NewDecoder(r.Body).Decode(&noteReq)
	if noteReq.SessionID == "" {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find the session with the matching session ID
	noteMutex.Lock()
	var userID uint32
	sessionFound := false
	for _, session := range sessions {
		if session.ID == noteReq.SessionID {
			sessionFound = true
			userID = session.UserID
			break
		}
	}
	noteMutex.Unlock()

	// If session not found, return unauthorized
	if !sessionFound {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate a unique ID for the note
	newNoteID := uint32(len(notes) + 1)

	// Add the note to the notes slice
	noteMutex.Lock()
	newNote := Note{
		ID:     newNoteID,
		Note:   noteReq.Note,
		UserID: userID,
	}
	notes = append(notes, newNote)
	noteMutex.Unlock()

	// Create the response
	response := struct {
		ID uint32 `json:"id"`
	}{
		ID: newNoteID,
	}

	// Set the response header and send the response as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a NoteRequest struct
	var noteReq struct {
		SessionID string `json:"sid"`
		ID        uint32 `json:"id"`
	}

	err := json.NewDecoder(r.Body).Decode(&noteReq)
	if noteReq.SessionID == "" && noteReq.ID == 0 {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find the session with the matching session ID
	noteMutex.Lock()
	sessionFound := false
	for _, session := range sessions {
		if session.ID == noteReq.SessionID {
			sessionFound = true

			// Find the note with the matching ID and user ID
			for j, note := range notes {
				if note.ID == noteReq.ID && note.UserID == session.UserID {
					// Remove the note from the notes slice
					notes = append(notes[:j], notes[j+1:]...)
					break
				}
			}
		}
	}
	noteMutex.Unlock()

	// If session not found, return unauthorized
	if !sessionFound {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(http.StatusOK)
}
