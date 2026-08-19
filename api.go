package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	AppAPIHost       = "https://app-api.pixiv.net"
	OAuthHost        = "https://oauth.secure.pixiv.net"
	ClientID         = "MOBrBDS8blbauoSck0ZfDbtuzpyT"
	ClientSecret     = "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj"
	HashSecret       = "28c1fdd170a5204386cb1313c7077b34f83e4aaf4aa829ce78c231e05b0bae2c"
	DefaultUserAgent = "PixivIOSApp/7.13.3 (iOS 14.6; iPhone13,2)"
)

type PixivAPI struct {
	AccessToken  string
	RefreshToken string
	HttpClient   *http.Client
}

type AuthResponse struct {
	Response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID   any    `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
	} `json:"response"`
}

type ImageUrls struct {
	SquareMedium string `json:"square_medium"`
	Medium       string `json:"medium"`
	Large        string `json:"large"`
}

type UserInfo struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Account string `json:"account"`
}

type Illust struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	ImageUrls      ImageUrls `json:"image_urls"`
	User           UserInfo  `json:"user"`
	PageCount      int       `json:"page_count"`
	MetaSinglePage struct {
		OriginalImageUrl string `json:"original_image_url"`
	} `json:"meta_single_page"`
	MetaPages []struct {
		ImageUrls ImageUrls `json:"image_urls"`
	} `json:"meta_pages"`
}
type IllustDetailResponse struct {
	Illust Illust `json:"illust"`
}
type IllustResponse struct {
	Illusts []Illust `json:"illusts"`
	NextURL string   `json:"next_url"`
}

func NewPixivAPI(accessToken, refreshToken string) *PixivAPI {
	return &PixivAPI{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		HttpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *PixivAPI) genHash() (string, string) {
	localTime := time.Now().UTC().Format("2006-01-02T15:04:05+00:00")
	hash := md5.Sum([]byte(localTime + HashSecret))
	return localTime, hex.EncodeToString(hash[:])
}

// 1. Token Refresh 功能
func (a *PixivAPI) Auth(refreshToken string) (*AuthResponse, error) {
	localTime, hash := a.genHash()
	data := url.Values{}
	data.Set("get_secure_url", "1")
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", OAuthHost+"/auth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Client-Time", localTime)
	req.Header.Set("X-Client-Hash", hash)
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("App-OS", "ios")
	req.Header.Set("App-OS-Version", "14.6")

	resp, err := a.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth failed: %s", string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	a.AccessToken = authResp.Response.AccessToken
	a.RefreshToken = authResp.Response.RefreshToken

	return &authResp, nil
}

func (a *PixivAPI) request(method, path string, params url.Values) (*http.Response, error) {
	u, _ := url.Parse(AppAPIHost + path)
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("App-OS", "ios")
	req.Header.Set("App-OS-Version", "14.6")
	req.Header.Set("Authorization", "Bearer "+a.AccessToken)

	resp, err := a.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// 如果返回 401 且有 RefreshToken，尝试自动刷新并重试一次
	if resp.StatusCode == http.StatusUnauthorized && a.RefreshToken != "" {
		_, err := a.Auth(a.RefreshToken)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+a.AccessToken)
			return a.HttpClient.Do(req)
		}
	}

	return resp, nil
}

// 2. UserIllusts 功能
func (a *PixivAPI) UserIllusts(userID int, illustType string, offset int) (*IllustResponse, error) {
	params := url.Values{}
	params.Set("user_id", fmt.Sprint(userID))
	params.Set("type", illustType)
	params.Set("filter", "for_ios")
	if offset > 0 {
		params.Set("offset", fmt.Sprint(offset))
	}

	resp, err := a.request("GET", "/v1/user/illusts", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result IllustResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// 3. IllustRanking 功能
func (a *PixivAPI) IllustRanking(mode string, date string, offset int) (*IllustResponse, error) {
	params := url.Values{}
	params.Set("mode", mode)
	params.Set("filter", "for_ios")
	if date != "" {
		params.Set("date", date)
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprint(offset))
	}

	resp, err := a.request("GET", "/v1/illust/ranking", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result IllustResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// 4. Download 功能
func (a *PixivAPI) Download(imageUrl string) ([]byte, error) {
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Referer", "https://app-api.pixiv.net/")

	resp, err := a.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return body, nil
}

// 5. IllustDetail 功能
func (a *PixivAPI) IllustDetail(illustID int) (*IllustDetailResponse, error) {
	params := url.Values{}
	params.Set("illust_id", fmt.Sprint(illustID))

	resp, err := a.request("GET", "/v1/illust/detail", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result IllustDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
