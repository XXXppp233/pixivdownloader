package main

import (
	"fmt"
	"log"
	"os"
)

var (
	authorization string
	accessToken   string
	refreshToken  string
	api           *PixivAPI
	RankMode      map[string]bool
)

func init() {
	authorization = os.Getenv("AUTHORIZATION")
	if authorization == "" {
		log.Fatal("missing authorization")
	}
	accessToken = "accessToken"
	refreshToken = os.Getenv("REFRESH_TOKEN")
	if refreshToken == "" {
		log.Fatal("Missing Refresh Token")
	}
	api = NewPixivAPI(accessToken, refreshToken)
	fmt.Println("Refresh Token")
	result, err := api.Auth(refreshToken)
	if err != nil {
		log.Fatalf("Auth failed: %v", err)
	}
	fmt.Println("Login As", result.Response.User.Name)

	RankMode = make(map[string]bool)
	for _, mode := range []string{"day", "week", "month", "day_male", "day_female", "week_original", "week_rookie", "day_manga"} {
		RankMode[mode] = true
	}

	fmt.Println("Init Complete: Access Token and Refresh Token are set.")
}
