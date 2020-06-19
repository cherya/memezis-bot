package memezis_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type Client struct {
	serverURL string
	authToken string
	hc        *http.Client
}

func NewClient(url string, token string) *Client {
	c := &http.Client{
		Timeout: time.Minute,
	}
	return &Client{
		serverURL: url,
		hc:        c,
		authToken: token,
	}
}

const (
	SourceMemezisBot = "memezis_bot"
)

const (
	PublishStatusEnqueued = "enqueued"
	PublishStatusDeclined = "declined"
)

type UploadMediaResponse struct {
	Filename string `json:"filename"`
}

func (c *Client) UploadMedia(file io.Reader, filename string) (*UploadMediaResponse, error) {
	var requestBody bytes.Buffer
	multiPartWriter := multipart.NewWriter(&requestBody)

	fileWriter, err := multiPartWriter.CreateFormFile("file", filename)
	if err != nil {
		return nil, errors.Wrap(err, "UploadMedia: can't create form file")
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, errors.Wrap(err, "UploadMedia: can't copy file to writer")
	}

	multiPartWriter.Close()

	req, err := c.request(http.MethodPost, "upload", &requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "UploadMedia: can't create request")
	}
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

	response, err := c.do(req)
	if err != nil {
		return nil, errors.Wrap(err, "UploadMedia: can't do request")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("UploadMedia: status not ok %d", response.StatusCode)
	}

	var res UploadMediaResponse
	_ = json.NewDecoder(response.Body).Decode(&res)
	return &res, nil
}

type Media struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	SourceID string `json:"source_id,omitempty"`
	SHA1     string `json:"sha_1"`
}

type addPostRequest struct {
	Media   []Media  `json:"media"`
	AddedBy string   `json:"added_by"`
	Text    string   `json:"text"`
	Tags    []string `json:"tags"`
}

type addPostResponse struct {
	ID         int64   `json:"id"`
	Duplicates []int64 `json:"duplicates"`
}

func (c *Client) AddPost(media []Media, addedBy, text string, tags []string) (*addPostResponse, error) {
	r := addPostRequest{
		Media:   media,
		AddedBy: addedBy,
		Text:    text,
		Tags:    tags,
	}
	body, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "AddPost: can't marshal request")
	}
	req, err := c.request(http.MethodPost, "post/add", bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "AddPost: can't build request")
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, errors.Wrap(err, "AddPost: can't do request")
	}
	defer resp.Body.Close()

	var res addPostResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)

	return &res, nil
}

type Votes struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type GetPostByIDResponse struct {
	ID      int64    `json:"id"`
	Media   []Media  `json:"media"`
	AddedBy string   `json:"added_by"`
	Source  string   `json:"source"`
	Text    string   `json:"text"`
	Tags    []string `json:"tags"`
	Votes   Votes    `json:"Votes"`
}

func (c *Client) GetPost(postID int64) (*GetPostByIDResponse, error) {
	req, err := c.request(http.MethodGet, fmt.Sprintf("post/%d", postID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetPost: can't build request")
	}
	resp, err := c.do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "GetPost: can't do request")
	}

	var res GetPostByIDResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, errors.Wrap(err, "GetPost: decode response")
	}

	return &res, nil
}

type voteRequest struct {
	UserID string `json:"user_id"`
}

type VoteResponse struct {
	Up     int64  `json:"up"`
	Down   int64  `json:"down"`
	Status string `json:"status"`
}

var voteToPath = map[bool]string{
	true:  "upvote",
	false: "downvote",
}

func (c *Client) VoteByPostID(postID int64, userID int, isUp bool) (*VoteResponse, error) {
	r := voteRequest{
		UserID: strconv.Itoa(userID),
	}
	body, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "VoteByPostID: can't marshal request")
	}
	path := fmt.Sprintf("post/%d/%s", postID, voteToPath[isUp])
	req, err := c.request(http.MethodPost, path, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "VoteByPostID: can't build request")
	}
	resp, err := c.do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "VoteByPostID: can't do request")
	}

	var res VoteResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)

	return &res, nil
}

type publishRequest struct {
	PublishedAt int64  `json:"published_at"`
	PublishedTo string `json:"published_to"`
}

func (c *Client) PublishPost(postID int64, publishedAt time.Time, publishedTo string) error {
	r := publishRequest{
		PublishedAt: publishedAt.Unix(),
		PublishedTo: publishedTo,
	}
	body, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "PublishPost: can't marshal request")
	}
	path := fmt.Sprintf("post/%d/publish", postID)
	req, err := c.request(http.MethodPost, path, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "PublishPost: can't build request")
	}
	resp, err := c.do(req)
	defer resp.Body.Close()
	if err != nil {
		return errors.Wrap(err, "PublishPost: can't do request")
	}

	return nil
}

type QueueInfoResponse struct {
	Length       int64     `json:"length"`
	LastPostTime time.Time `json:"last_post_time"`
	DueTime      time.Time `json:"due_time"`
}

func (c *Client) QueueInfo(queue string) (*QueueInfoResponse, error) {
	path := fmt.Sprintf("queue/%s/info", queue)
	req, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "QueueInfo: can't build request")
	}
	resp, err := c.do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "QueueInfo: can't do request")
	}

	var res QueueInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, errors.Wrap(err, "QueueInfo: can't decode response")
	}

	return &res, nil
}

func (c *Client) request(method, path string, body *bytes.Buffer) (*http.Request, error) {
	if body == nil {
		body = bytes.NewBufferString("")
	}
	req, err := http.NewRequest(method, c.serverURL+path, body)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", c.authToken)
	return c.hc.Do(req)
}
