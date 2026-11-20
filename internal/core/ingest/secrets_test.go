package ingest

import "testing"

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
