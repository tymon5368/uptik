package platforms

import (
	"fmt"
	"strings"
)

var checkpointIndicators = []string{
	"too many requests",
	"action blocked",
	"security check",
	"verify it's you",
	"unusual activity",
	"tài khoản bị hạn chế",
	"thao tác quá nhanh",
	"xác minh bảo mật",
	"captcha",
}

// CheckCircuitBreaker inspects page content for checkpoint or ban triggers
func CheckCircuitBreaker(pageContent string) error {
	lower := strings.ToLower(pageContent)
	for _, ind := range checkpointIndicators {
		if strings.Contains(lower, ind) {
			return fmt.Errorf("🚨 [CIRCUIT BREAKER KÍCH HOẠT] Phát hiện cảnh báo an toàn từ nền tảng: %q. Hệ thống đã tự động dừng để bảo vệ tài khoản!", ind)
		}
	}
	return nil
}
