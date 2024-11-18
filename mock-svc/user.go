package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"svc-opt/lib/model"
	"time"
)

// GetUsers handles requests to fetch one or more users by their IDs.
func GetUsers(w http.ResponseWriter, r *http.Request) {
	// Extract user IDs from the URL
	userIDs := strings.TrimPrefix(r.URL.Path, "/user/")
	idList := strings.Split(userIDs, ",")

	var foundUsers []*model.User
	for _, idStr := range idList {
		userID, err := strconv.Atoi(idStr)
		if err != nil || userID < 1 {
			http.Error(w, "Invalid user ID: "+idStr, http.StatusBadRequest)
			return
		}

		// Search for the user in the simulated database
		id, _ := strconv.ParseInt(idStr, 10, 64)
		foundUsers = append(foundUsers, model.MockUser(int(id)))
	}

	time.Sleep(300 * time.Millisecond)

	if len(foundUsers) == 0 {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foundUsers)
}

func main() {
	http.HandleFunc("/user/", GetUsers)
	log.Println("Starting user service on :8081")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
