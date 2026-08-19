package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	http.HandleFunc("/", DefaultHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != authorization {
		return
	}
	params := r.URL.Query()
	typ := params.Get("type")
	switch typ {
	case "user":
		HandleUser(w, r)
	case "image":
		HandleImage(w, r)
	case "rank":
		HandleRank(w, r)
	default:
		return
	}
}

func HandleUser(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	id := params.Get("id")
	if id == "" {
		fmt.Fprintf(w, "Error: Missing user ID")
		return
	}
	userID, err := strconv.Atoi(id)
	if err != nil {
		fmt.Fprintf(w, "Error: Invalid user ID")
		return
	}
	illust, err := api.UserIllusts(userID, "illust", 0)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	data := make(map[int]string)
	for _, v := range illust.Illusts {
		data[v.ID] = v.Title
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func HandleImage(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	id := params.Get("id")
	if id == "" {
		fmt.Fprintf(w, "Error: Missing image ID")
		return
	}
	imageID, err := strconv.Atoi(id)
	if err != nil {
		fmt.Fprintf(w, "Error: Invalid image ID")
		return
	}
	detail, err := api.IllustDetail(imageID)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	body, err := api.Download(detail.Illust.ImageUrls.Large)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(body)
}

type SimpleRank struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	User   string `json:"user"`
	UserID int    `json:"userid"`
}

func HandleRank(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	mode := params.Get("mode")
	if !RankMode[mode] {
		fmt.Fprintf(w, "Error: Invalid mode")
		return
	}
	result, err := api.IllustRanking(mode, "", 0)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	sr := make([]SimpleRank, len(result.Illusts))
	for i, v := range result.Illusts {
		sr[i] = SimpleRank{
			ID:     v.ID,
			Title:  v.Title,
			User:   v.User.Name,
			UserID: v.User.ID,
		}
	}
	data, _ := json.Marshal(sr)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
