package bot

import (
	"fmt"
	"github.com/cherya/memezis/pkg/memezis"
	"math/rand"
	"strings"
)

var (
	successTexts = []string{
		"принято",
		"найс",
		"кайф",
		"спасибо",
		"ваш мем очень важен для нас",
		"надеюсь это не боян",
		"посмотрим...",
		"передал куда следует.",
	}
	votingTexts       = []string{"🌚", "💅🏻", "👆🏼", "💪🏾", "🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🦝", "🐻", "🐼", "🦘", "🦡", "🐨", "🐯", "🦁", "🐮", "🐷", "🐽", "🐸", "🐵", "🙈", "🙉", "🙊", "🐒", "🐔", "🐧", "🐦", "🐤", "🐣", "🐥", "🦆", "🦢", "🦅", "🦉", "🦚", "🦜", "🦇", "🐺", "🐗", "🐴", "🦄", "🐝", "🐛", "🦋", "🐌", "🐚", "🐞", "🐜", "🦗", "🕷", "🕸", "🦂", "🦟", "🦠", "🐢", "🐍", "🦎", "🦖", "🦕", "🐙", "🦑", "🦐", "🦀", "🐡", "🐠", "🐟", "🐬", "🐳", "🐋", "🦈", "🐊", "🐅", "🐆", "🦓", "🦍", "🐘", "🦏", "🦛", "🐪", "🐫", "🦙", "🦒", "🐃", "🐂", "🐄", "🐎", "🐖", "🐏", "🐑", "🐐", "🦌", "🐕", "🐩", "🐈", "🐓", "🦃", "🕊", "🐇", "🐁", "🐀", "🐿", "🦔", "🐾", "🐉", "🐲"}
	unsupportedText   = "Я понимаю только фото, видео и гифки. Если есть предложения – пиши @cherya"
	banText           = "тебя даже бот с мемами забанил, пиздец"
	voteCallbackTexts = []string{"голос учтен", "постчитано", "запомнил", "как скажешь", "я не согласен, но ладно", "ок", "допустим", "записал"}
	alreadyVotedTexts = []string{"ну ты дурак?", "че ты жмешь?", "да уже", "???", "#$%&@??"}
	queuedText        = "уже в очереди"
	declinedText      = "уже выкинул"

	completeDuplicateTexts = []string{"сто процентов абсолютно точно боян", "БОЯН!!!", "Дед, таблетки"}
	likelyDuplicateTexts = []string{"скорее всего это уже было"}
)

func getVoteCallbackTexts() string {
	return randomText(voteCallbackTexts)
}

func getAlreadyVotedCallbackTexts() string {
	return randomText(alreadyVotedTexts)
}

func getSuccessText() string {
	return randomText(successTexts)
}

func getVotingText() string {
	return randomText(votingTexts)
}

func randomText(t []string) string {
	return t[rand.Intn(len(t))]
}

func getDuplicatesLinks(posts []*memezis.Post) []string {
	var links []string
	for _, c := range posts {
		if c.SourceURL != "" {
			links = append(links, c.SourceURL)
		} else {
			for _, p := range c.Publish {
				links = append(links, p.URL)
			}
		}
	}
	return links
}

func getDuplicatesText(d *memezis.FindDuplicatesByPostIDResponse) string {
	msg := strings.Builder{}
	dupls := d.Likely
	if len(d.Complete) > 0 {
		dupls = d.Complete
	}
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
