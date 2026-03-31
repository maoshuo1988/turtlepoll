package captchaimage

import "testing"

func TestGenerateAndVerify(t *testing.T) {
	id, b64, err := Generate()
	if err != nil {
		t.Fatalf("Generate() err: %v", err)
	}
	if id == "" {
		t.Fatalf("Generate() id empty")
	}
	if b64 == "" {
		t.Fatalf("Generate() base64 empty")
	}

	// 拿不到答案时，至少保证“错误答案”一定校验失败
	if Verify(id, "wrong") {
		t.Fatalf("Verify() should be false for wrong answer")
	}
}
