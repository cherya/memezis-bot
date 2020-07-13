package dailyword

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

type WordGenerator struct {
	redis *redis.Pool
}

func NewWordGenerator(pool *redis.Pool) *WordGenerator {
	return &WordGenerator{redis: pool}
}

func (g *WordGenerator) Get(uid string) (string, error) {
	conn := g.redis.Get()
	defer conn.Close()

	key := fmt.Sprintf("dailyword:%s:%s", uid, time.Now().Format("02.01.2006"))

	gen, err := redis.String(conn.Do("GET", key))
	if err != nil {
		if err != redis.ErrNil {
			return "", errors.Wrap(err, "can't get value from redis")
		}
	}

	if gen != "" {
		return gen, nil
	}

	gen, err = getWords()
	if err != nil {
		return "", errors.Wrap(err, "can't get words")
	}

	d, _ := time.ParseDuration("24h")
	_ = conn.Send("MULTI")
	_ = conn.Send("SETNX", key, gen)
	_ = conn.Send("EXPIRE", key, d.Seconds())
	_, err = conn.Do("EXEC")
	if err != nil {
		return "", errors.Wrap(err, "can't set value to redis")
	}

	return gen, nil
}

const randomURL = "https://ru.wikipedia.org/w/api.php?format=json&action=query&generator=random&grnnamespace=0&grnlimit=1&prop=revisions|images|categories&rvprop=content&rvslots=*"
const infoURL = "https://en.wikipedia.org/w/api.php?action=query&prop=info&pageids=%s&inprop=url"

var excludeCategories = []string{
	"Биографии",
	"Персоналии",
	"Родившиеся",
	"Однофамильцы",
	"Населённые пункты",
	"Компании",
}

var removeValues = []string{
	"(значения)",
}

func getWords() (string, error) {
	resp, err := getOne()
	if err != nil {
		return "", err
	}
	jsonResp := string(resp)

	title := gjson.Get(jsonResp, "query.pages.*.title").String()
	categories := gjson.Get(jsonResp, "query.pages.*.categories").Array()
	for _, c := range categories {
		cTitle := c.Get("title").String()
		for _, e := range excludeCategories {
			if strings.Contains(cTitle, e) {
				return getWords()
			}
		}
	}

	for _, r := range removeValues {
		title = strings.Replace(title, r, "", -1)
	}

	title = trimBrackets(title)

	return title, nil
}

func trimBrackets(s string) string {
	s = strings.TrimSpace(s)
	if strings.LastIndex(s, ")") == len(s)-1 {
		idx := strings.LastIndex(s, "(")
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	return s
}

func getOne() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, randomURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "can't build request")
	}

	c := http.Client{}

	resp, err :=c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "can't do request")
	}

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "can't read response body")
		}
		return bodyBytes, nil
	}


	return nil, errors.Wrapf(err, "response code not ok %d", resp.StatusCode)
}

func getPageUrl(pageID string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(infoURL, pageID), nil)
	if err != nil {
		return "", errors.Wrap(err, "can't build info request")
	}

	c := http.Client{}

	resp, err :=c.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "can't do info request")
	}

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "can't read info response body")
		}
		body := string(bodyBytes)

		path := fmt.Sprintf("query.pages.%s.fullurl", pageID)
		return gjson.Get(body, path).String(), nil
	}

	return "", errors.Wrapf(err, "info response code not ok %d", resp.StatusCode)
}