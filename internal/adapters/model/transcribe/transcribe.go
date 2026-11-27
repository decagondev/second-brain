// Package transcribe turns recordings into transcripts (SB-6102; sb-cell(5),
// model.transcribe): a whisper.cpp server (POST /inference, the transcribe profile on a
// self-hosted box) or an OpenAI-compatible provider (POST /v1/audio/transcriptions). Both
// answer verbose JSON with timed segments. Standard library only.
package transcribe

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// Client is one transcription provider.
type Client struct {
	// Provider is whispercpp or openaicompat.
	Provider string
	Base     string // http://transcribe:8080, or https://api.example/v1
	Model    string // openaicompat's model id
	// Key is the provider's key at call time; nil, or "", sends none (a server on the box).
	Key  func(ctx context.Context) (string, error)
	HTTP *http.Client
}

var _ ports.Transcriber = Client{}

// Error is a provider's refusal by status, never its body.
type Error struct{ Status int }

func (e Error) Error() string { return fmt.Sprintf("transcribe: HTTP %d", e.Status) }

// HTTPStatus implements ports.StatusError.
func (e Error) HTTPStatus() int { return e.Status }

type verbose struct {
	Text     string  `json:"text"`
	Duration float64 `json:"duration"`
	Segments []struct {
		Start   float64 `json:"start"`
		End     float64 `json:"end"`
		Text    string  `json:"text"`
		Speaker string  `json:"speaker"`
	} `json:"segments"`
}

// Transcribe implements ports.Transcriber.
func (c Client) Transcribe(ctx context.Context, p ports.Part) ([]ports.Segment, float64, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	f, err := w.CreateFormFile("file", p.Name)
	if err != nil {
		return nil, 0, err
	}
	if _, err := f.Write(p.Data); err != nil {
		return nil, 0, err
	}
	_ = w.WriteField("response_format", "verbose_json")
	u := strings.TrimSuffix(c.Base, "/")
	switch c.Provider {
	case "whispercpp":
		u += "/inference"
	case "openaicompat":
		_ = w.WriteField("model", c.Model)
		_ = w.WriteField("timestamp_granularities[]", "segment")
		u += "/audio/transcriptions"
	default:
		return nil, 0, fmt.Errorf("transcribe: provider %q", c.Provider)
	}
	if err := w.Close(); err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c.Key != nil {
		k, err := c.Key(ctx)
		if err != nil {
			return nil, 0, err
		}
		if k != "" {
			req.Header.Set("Authorization", "Bearer "+k)
		}
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("transcribe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return nil, 0, Error{Status: resp.StatusCode}
	}
	var v verbose
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&v); err != nil {
		return nil, 0, errors.New("transcribe: an answer that isn't verbose JSON")
	}
	var segs []ports.Segment
	end := 0.0
	for _, s := range v.Segments {
		if t := strings.TrimSpace(s.Text); t != "" {
			segs = append(segs, ports.Segment{Start: s.Start, Speaker: s.Speaker, Text: t})
		}
		end = max(end, s.End)
	}
	if len(segs) == 0 && strings.TrimSpace(v.Text) != "" {
		segs = []ports.Segment{{Text: strings.TrimSpace(v.Text)}}
	}
	minutes := max(v.Duration, end) / 60
	return segs, minutes, nil
}

// Text is a transcript as the lines ingest reads: "[HH:MM:SS] Speaker: text", one a segment.
func Text(segs []ports.Segment) string {
	var b strings.Builder
	for _, s := range segs {
		t := int(s.Start)
		fmt.Fprintf(&b, "[%02d:%02d:%02d] ", t/3600, t/60%60, t%60)
		if s.Speaker != "" {
			b.WriteString(s.Speaker + ": ")
		}
		b.WriteString(s.Text + "\n")
	}
	return strings.TrimSpace(b.String())
}

// Minutes is about how long a recording runs, before it is sent: exact for a WAV file from
// its header, otherwise from its size at 128 kbit/s, which overestimates most speech.
func Minutes(p ports.Part) float64 {
	b := p.Data
	if len(b) >= 44 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WAVE" {
		// find the fmt chunk's byte rate and the data chunk's size
		rate, size := 0, 0
		for i := 12; i+8 <= len(b); {
			id, n := string(b[i:i+4]), int(binary.LittleEndian.Uint32(b[i+4:]))
			switch {
			case id == "fmt " && i+20 <= len(b):
				rate = int(binary.LittleEndian.Uint32(b[i+16:]))
			case id == "data":
				size = n
			}
			if id == "data" || n < 0 {
				break
			}
			i += 8 + n + n%2
		}
		if rate > 0 && size > 0 {
			return float64(size) / float64(rate) / 60
		}
	}
	return float64(len(b)) / 16000 / 60
}
