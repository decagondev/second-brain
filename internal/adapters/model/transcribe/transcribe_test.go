package transcribe

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// SB-6102: whisper.cpp and an OpenAI-compatible provider each get the file as multipart and
// answer timed segments; a refusal is its status only; a WAV's length is read from its
// header.
func TestTranscribe(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if err := r.ParseMultipartForm(1 << 20); err != nil || r.FormValue("response_format") != "verbose_json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path == "/v1/audio/transcriptions" && (r.Header.Get("Authorization") != "Bearer k1" || r.FormValue("model") != "whisper-1") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "hello all", "duration": 90.0,
			"segments": []map[string]any{{"start": 0.0, "end": 2.0, "text": " Hello all."}, {"start": 65.5, "end": 90.0, "text": "The close moves to Tuesday."}}})
	}))
	t.Cleanup(srv.Close)
	part := ports.Part{Name: "standup.wav", Data: []byte("audio")}
	segs, min, err := Client{Provider: "whispercpp", Base: srv.URL, HTTP: srv.Client()}.Transcribe(context.Background(), part)
	if err != nil || len(segs) != 2 || min != 1.5 {
		t.Fatalf("whispercpp: %v %+v %v", err, segs, min)
	}
	if got := Text(segs); got != "[00:00:00] Hello all.\n[00:01:05] The close moves to Tuesday." {
		t.Fatalf("text: %q", got)
	}
	key := func(context.Context) (string, error) { return "k1", nil }
	if _, _, err := (Client{Provider: "openaicompat", Base: srv.URL + "/v1", Model: "whisper-1", Key: key, HTTP: srv.Client()}).Transcribe(context.Background(), part); err != nil {
		t.Fatal(err)
	}
	bad := func(context.Context) (string, error) { return "nope", nil }
	_, _, err = Client{Provider: "openaicompat", Base: srv.URL + "/v1", Model: "whisper-1", Key: bad, HTTP: srv.Client()}.Transcribe(context.Background(), part)
	var e Error
	if err == nil || !strings.Contains(err.Error(), "401") || !errors.As(err, &e) || e.HTTPStatus() != 401 {
		t.Fatalf("refusal: %v", err)
	}
	if paths[0] != "/inference" || paths[1] != "/v1/audio/transcriptions" {
		t.Fatalf("paths: %v", paths)
	}
	// A WAV of 16 kHz mono 16-bit (32000 bytes a second) holding 3 minutes.
	wav := make([]byte, 44)
	copy(wav, "RIFF")
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint32(wav[28:], 32000)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], 32000*180)
	if m := Minutes(ports.Part{Data: wav}); m != 3 {
		t.Fatalf("wav minutes: %v", m)
	}
	if m := Minutes(ports.Part{Data: make([]byte, 16000*60*2)}); m != 2 {
		t.Fatalf("estimated minutes: %v", m)
	}
}
