package utils

import (
	"context"
	"errors"
	"fmt"
)

func GetContextUserIdWithDefault(ctx context.Context) int64 {
	id, _ := GetContextUserId(ctx)
	return id
}

func GetContextUserId(ctx context.Context) (int64, error) {
	if ctx == nil {
		return 0, errors.New("上下文ctx为空")
	}

	v := ctx.Value("user_id")
	if v == nil {
		return 0, errors.New("user_id不存在于上下文ctx中")
	}

	id, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("user_id类型不是int64，实际类型为%T", v)
	}

	return id, nil
}
