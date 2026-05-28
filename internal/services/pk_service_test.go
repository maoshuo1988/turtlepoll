package services

import (
	"encoding/json"
	"testing"
)

func TestPKBetFormUnmarshalJSONAcceptsStringTopicId(t *testing.T) {
	var form PKBetForm
	if err := json.Unmarshal([]byte(`{"topicId":"1","side":"B","requestId":"req-1","amount":100}`), &form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.TopicId != 1 {
		t.Fatalf("expected topicId 1, got %d", form.TopicId)
	}
	if form.Side != "B" {
		t.Fatalf("expected side B, got %q", form.Side)
	}
	if form.RequestId != "req-1" {
		t.Fatalf("expected requestId req-1, got %q", form.RequestId)
	}
}

func TestPKBetFormUnmarshalJSONAcceptsNumberTopicId(t *testing.T) {
	var form PKBetForm
	if err := json.Unmarshal([]byte(`{"topicId":1,"side":"A","requestId":"req-2"}`), &form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.TopicId != 1 {
		t.Fatalf("expected topicId 1, got %d", form.TopicId)
	}
}
