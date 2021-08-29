package bot

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/cherya/memezis/pkg/memezis"
)

type textType string

const (
	TextTypeSucessUpload textType = "sucess"
	TextTypeVoting       textType = "voting"
	TextTypeUnsupported  textType = "unsupported"
	TextTypeBanned       textType = "banned"
	TextTypeVoteCallback textType = "vote_callback"
	TextTypeVotedAlready textType = "voted_already"
	TextTypeQueued       textType = "queued"
	TextTypeDeclined     textType = "declined"
	TextTypeOwnPostVote  textType = "own_post_vote"
)

var texts = map[textType][]string{
	TextTypeSucessUpload: {"принято", "найс", "кайф", "спасибо", "ваш мем очень важен для нас", "надеюсь это не боян", "посмотрим...", "передал куда следует."},
	TextTypeVoting:       {"🌚", "💅🏻", "👆🏼", "💪🏾", "🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🦝", "🐻", "🐼", "🦘", "🦡", "🐨", "🐯", "🦁", "🐮", "🐷", "🐽", "🐸", "🐵", "🙈", "🙉", "🙊", "🐒", "🐔", "🐧", "🐦", "🐤", "🐣", "🐥", "🦆", "🦢", "🦅", "🦉", "🦚", "🦜", "🦇", "🐺", "🐗", "🐴", "🦄", "🐝", "🐛", "🦋", "🐌", "🐚", "🐞", "🐜", "🦗", "🕷", "🕸", "🦂", "🦟", "🦠", "🐢", "🐍", "🦎", "🦖", "🦕", "🐙", "🦑", "🦐", "🦀", "🐡", "🐠", "🐟", "🐬", "🐳", "🐋", "🦈", "🐊", "🐅", "🐆", "🦓", "🦍", "🐘", "🦏", "🦛", "🐪", "🐫", "🦙", "🦒", "🐃", "🐂", "🐄", "🐎", "🐖", "🐏", "🐑", "🐐", "🦌", "🐕", "🐩", "🐈", "🐓", "🦃", "🕊", "🐇", "🐁", "🐀", "🐿", "🦔", "🐾", "🐉", "🐲"},
	TextTypeUnsupported:  {"Я понимаю только фото, видео и гифки. Если есть предложения – пиши @cherya"},
	TextTypeBanned:       {"тебя даже бот с мемами забанил, пиздец"},
	TextTypeVoteCallback: {"голос учтен", "постчитано", "запомнил", "как скажешь", "я не согласен, но ладно", "ок", "допустим", "записал"},
	TextTypeVotedAlready: {"ну ты дурак?", "че ты жмешь?", "да уже", "???", "#$%&@??"},
	TextTypeQueued:       {"уже в очереди"},
	TextTypeDeclined:     {"уже выкинул"},
	TextTypeOwnPostVote:  {"не считается", "неа", "хватит дрочить"},
}
var (
	likelyDuplicateTexts = []string{"скорее всего это уже было"}
)

func getText(tt textType) string {
	return randomText(texts[tt])
}

func randomText(t []string) string {
	return t[rand.Intn(len(t))]
}

func getDuplicatesLinks(duplicates []*memezis.PostDuplicate) []string {
	var links []string
	for _, c := range duplicates {
		post := c.GetPost()
		if post.GetSourceURL() != "" {
			if post.GetSource() != "" {
				links = append(links, fmt.Sprintf("[%s](%s)", post.GetSource(), post.GetSource()))
			} else {
				links = append(links, post.GetSourceURL())
			}
		} else {
			for _, p := range post.GetPublish() {
				links = append(links, p.URL)
			}
		}
	}
	return links
}

func getDuplicatesText(d *memezis.FindDuplicatesByPostIDResponse) string {
	msg := strings.Builder{}
	dupls := d.GetDuplicate()
	links := getDuplicatesLinks(dupls)
	msg.Write([]byte(randomText(likelyDuplicateTexts)))
	if len(links) != 0 {
		msg.Write([]byte("\nсурс:\n"))
		for _, l := range links {
			msg.Write([]byte(fmt.Sprintf("\n%s", l)))
		}
	}
	return msg.String()
}
