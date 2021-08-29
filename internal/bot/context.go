package bot

import (
	"context"
	"strconv"
)

type botCtxKey string

func (c botCtxKey) String() string {
	return string(c)
}

var (
	contextKeyUserID = botCtxKey("user_id")
)

func setUserToContext(ctx context.Context, userID int) context.Context {
	val := strconv.FormatInt(int64(userID), 10)
	return context.WithValue(ctx, contextKeyUserID, val)
}

func userIDFromContext(ctx context.Context) string {
	return ctx.Value(contextKeyUserID).(string)
}
