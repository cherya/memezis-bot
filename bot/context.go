package bot

import "context"

type botCtxKey string

func (c botCtxKey) String() string {
	return string(c)
}

var (
	contextKeyUserID = botCtxKey("user_id")
)

func setUserToContext(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

func userFromContext(ctx context.Context) int {
	return ctx.Value(contextKeyUserID).(int)
}
