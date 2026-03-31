package captchaimage

import (
	"image/color"

	"github.com/dchest/captcha"
	"github.com/mojocn/base64Captcha"
)

const (
	// ProtocolBase64Image 表示 base64Captcha 图片验证码的协议号。
	// 与现有：1=dchest, 2=rotate 保持兼容。
	ProtocolBase64Image = 3
)

// 默认使用内存存储（单机/开发最简单）；如需多实例，请替换为 redis store。
var store = base64Captcha.DefaultMemStore

// Generate 生成一张图片验证码，返回 captchaId 与 PNG 的 base64（不带 dataURL 前缀）。
func Generate() (captchaId string, captchaBase64 string, err error) {
	// 4 位字符验证码，图片尺寸与 dchest 接近，方便前端替换
	driver := base64Captcha.NewDriverString(
		captcha.StdHeight,
		captcha.StdWidth,
		0,
		base64Captcha.OptionShowHollowLine,
		4,
		"1234567890abcdefghijklmnopqrstuvwxyz",
		&color.RGBA{R: 255, G: 255, B: 255, A: 255},
		base64Captcha.DefaultEmbeddedFonts,
		nil,
	)
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64, _, genErr := c.Generate()
	if genErr != nil {
		return "", "", genErr
	}
	return id, b64, nil
}

// Verify 校验并在校验后清除（无论成功/失败），避免重复提交。
func Verify(captchaId string, captchaCode string) bool {
	return store.Verify(captchaId, captchaCode, true)
}
