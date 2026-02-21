package ssh

import "testing"

func TestLooksLikePasswordPrompt(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   bool
	}{
		{name: "empty accepted", prompt: "", want: true},
		{name: "password prompt", prompt: "user@example.com's password:", want: true},
		{name: "passphrase prompt", prompt: "Enter passphrase for key:", want: true},
		{name: "host key prompt rejected", prompt: "Are you sure you want to continue connecting (yes/no/[fingerprint])?", want: false},
		{name: "otp prompt rejected", prompt: "Verification code:", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikePasswordPrompt(tc.prompt); got != tc.want {
				t.Fatalf("looksLikePasswordPrompt(%q)=%v want %v", tc.prompt, got, tc.want)
			}
		})
	}
}
