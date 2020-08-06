package bot

import (
	"math/rand"
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

func getDuplicatesText(d *Duplicates) string {
	if len(d.Complete) > 0 {
		return randomText(completeDuplicateTexts)
	}
	if len(d.Likely) > 0 {
		return randomText(likelyDuplicateTexts)
	}
	return ""
}
