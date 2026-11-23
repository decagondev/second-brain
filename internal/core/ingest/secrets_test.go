package ingest

import (
	"strings"
	"testing"
)

// Test credentials are assembled at run time from parts, so the repository never holds a
// string that looks like a real key to a secret scanner (GitHub's push protection
// included). None of them is a real credential.
func fake(parts ...string) string {
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}

func TestSecretIn(t *testing.T) {
	for text, want := range map[string]string{
		fake("-----BEGIN OPENSSH ", "PRIVATE KEY-----\nb3Blbn"):             "private-key",
		fake("use AKIA", "ABCDEFGHIJKLMNOP for the bucket"):                 "aws-access-key",
		fake("token gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"):          "github-token",
		fake("SLACK=xo", "xb-123456789012-abcdefghij"):                      "slack-token",
		fake("sk", "_live_", "abcdefghijklmnopqrstuvwx"):                    "stripe-key",
		fake("key AI", "zaSyA1234567890abcdefghijklmnopqrstuv"):             "google-api-key",
		fake("db_pass", `word = "s3cr3t-pa55word-123"`):                     "assigned-secret",
		"Restart the consumer when the queue stalls.":                       "",
		"The password policy needs 12 characters.":                          "",
		"Set the api_key environment variable before running the importer.": "",
	} {
		if got := SecretIn(text); got != want {
			t.Errorf("SecretIn(%q) = %q, want %q", text, got, want)
		}
	}
}

// SB-1401: forms the first scanner missed. None of these is a real credential.
func TestSecretInMoreForms(t *testing.T) {
	for text, want := range map[string]string{
		fake("aws_secret_", "access_key = wJalrXUtnFEMI/", "K7MDENG/bPxRfiCYEXAMPLEKEY"):                         "assigned-secret",
		fake("AWS_SECRET_", "ACCESS_KEY=wJalrXUtnFEMI/", "K7MDENG/bPxRfiCYEXAMPLEKEY"):                           "assigned-secret",
		fake("Authorization: Bear", "er abcdefghijklmnopqrstuvwxyz0123456789"):                                   "bearer-token",
		fake("ey", "JhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"): "jwt",
		fake("postgres://ad", "min:S3cretPw@db.internal:5432/app"):                                               "url-credentials",
		fake("pass", `word = "correct horse battery staple"`):                                                    "assigned-secret",
		fake("tok", "en: 9f8e7d6c5b4a39281706f5e4d3c2b1a0"):                                                      "assigned-secret",
		fake("glp", "at-abcdefghijklmnopqrstu"):                                                                  "gitlab-token",
		fake("curl -H 'X: sb", "t_", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ'"):                             "sb-api-token",
		fake("sk-pr", "oj-abcdefghijklmnopqrstuvwxyz012345"):                                                     "model-api-key",
		fake("sk-an", "t-api03-abcdefghijklmnopqrstuvwxyz"):                                                      "model-api-key",
		"The token: rotate it monthly.":                                                                          "",
		"See https://docs.example.com/guide for the steps.":                                                      "",
	} {
		if got := SecretIn(text); got != want {
			t.Errorf("SecretIn(%q) = %q, want %q", text, got, want)
		}
	}
}

// SB-1401: a private key longer than a chunk is dropped from every chunk it touches.
func TestLongPrivateKeyIsDroppedEverywhere(t *testing.T) {
	body := ""
	for i := 0; i < 40; i++ {
		body += "MIIEowIBAAKCAQEAu1SU1LfVLPHCozMxH2Mo4lgOEePzNm0tRgeLezV20aZx\n"
	}
	text := "Deploy notes. " + fake("-----BEGIN RSA ", "PRIVATE KEY-----\n") + body + fake("-----END RSA ", "PRIVATE KEY-----") + " Restart the consumer."
	spans := SecretSpans(Normalise(text))
	if len(spans) == 0 {
		t.Fatal("no secret span found")
	}
	n := Normalise(text)
	for _, c := range Chunks(n) {
		chunk := string([]rune(n)[c.Start:c.End])
		if overlaps(spans, c.Start, c.End) == "" && SecretIn(chunk) == "" {
			if strings.Contains(chunk, "MIIEowIBAAKCAQEA") || strings.Contains(chunk, "PRIVATE") {
				t.Fatalf("a chunk with key material was kept: %q", chunk)
			}
		}
	}
	if len(Chunks(n)) < 2 {
		t.Fatalf("the key didn't span chunks (%d chunks); the test checks nothing", len(Chunks(n)))
	}
}
