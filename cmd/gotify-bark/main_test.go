package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDebugCompatibility(t *testing.T) {
	if os.Getenv("GOTIFY_BARK_TEST_CLI") == "1" {
		os.Args = append([]string{"gotify-bark"}, os.Args[3:]...)
		main()
		return
	}
	for _, tc := range []struct {
		name, env, flag, want string
	}{
		{"flag", "false", "--debug", "invalid Gotify URL"},
		{"environment", "true", "", "invalid Gotify URL"},
		{"invalid environment", "not-a-bool", "", "APP_DEBUG"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"-test.run=^TestDebugCompatibility$", "--", "--gotify-url=invalid", "--gotify-key=placeholder", "--shoutrrr-url=bark://:placeholder@localhost"}
			if tc.flag != "" {
				args = append(args, tc.flag)
			}
			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = append(os.Environ(), "GOTIFY_BARK_TEST_CLI=1", "APP_DEBUG="+tc.env)
			output, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(output), tc.want) {
				t.Fatalf("expected %q, got %s (error: %v)", tc.want, output, err)
			}
		})
	}
}
