package bot

import "math/rand"

var (
	successTexts = []string{
		"принято",
		"найс",
		"спс",
		"кайф",
		"спасибо",
		"ваш мем очень важен для нас",
	}
	votingTexts       = []string{"🌚", "💅🏻", "👆🏼", "💪🏾", "🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🦝", "🐻", "🐼", "🦘", "🦡", "🐨", "🐯", "🦁", "🐮", "🐷", "🐽", "🐸", "🐵", "🙈", "🙉", "🙊", "🐒", "🐔", "🐧", "🐦", "🐤", "🐣", "🐥", "🦆", "🦢", "🦅", "🦉", "🦚", "🦜", "🦇", "🐺", "🐗", "🐴", "🦄", "🐝", "🐛", "🦋", "🐌", "🐚", "🐞", "🐜", "🦗", "🕷", "🕸", "🦂", "🦟", "🦠", "🐢", "🐍", "🦎", "🦖", "🦕", "🐙", "🦑", "🦐", "🦀", "🐡", "🐠", "🐟", "🐬", "🐳", "🐋", "🦈", "🐊", "🐅", "🐆", "🦓", "🦍", "🐘", "🦏", "🦛", "🐪", "🐫", "🦙", "🦒", "🐃", "🐂", "🐄", "🐎", "🐖", "🐏", "🐑", "🐐", "🦌", "🐕", "🐩", "🐈", "🐓", "🦃", "🕊", "🐇", "🐁", "🐀", "🐿", "🦔", "🐾", "🐉", "🐲"}
	duplicateText     = "возможно это боян"
	unsupportedText   = "Я понимаю только фото, видео и гифки. Если есть предложения – пиши @cherya"
	banText           = "тебя даже бот с мемами забанил, пиздец"
	voteCallbackTexts = []string{"голос учтен", "постчитано", "запомнил", "как скажешь", "я не согласен, но ладно", "ок", "допустим", "записал"}
	alreadyVotedTexts = []string{"ну ты дурак?", "че ты жмешь?", "да уже", "???", "#$%&@??"}
	queuedText        = "уже в очереди"
	declinedText      = "уже выкинул"
)

func getVoteCallbackTexts() string {
	return voteCallbackTexts[rand.Intn(len(voteCallbackTexts))]
}

func getAlreadyVotedCallbackTexts() string {
	return alreadyVotedTexts[rand.Intn(len(alreadyVotedTexts))]
}

func getSuccessText() string {
	return successTexts[rand.Intn(len(successTexts))]
}

func getVotingText() string {
	return votingTexts[rand.Intn(len(votingTexts))]
}
