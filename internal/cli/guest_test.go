package cli

import (
	"reflect"
	"testing"
)

func TestBuildSSHArgs(t *testing.T) {
	cases := []struct {
		name  string
		user  string
		ip    string
		port  string
		id    string
		extra []string
		want  []string
	}{
		{"defaults", "me", "10.0.0.5", "22", "", nil, []string{"me@10.0.0.5"}},
		{"port", "me", "10.0.0.5", "2222", "", nil, []string{"-p", "2222", "me@10.0.0.5"}},
		{"identity", "root", "10.0.0.5", "22", "/k/id_ed25519", nil,
			[]string{"-i", "/k/id_ed25519", "root@10.0.0.5"}},
		{"extras before destination", "me", "10.0.0.5", "22", "", []string{"-v", "-o", "StrictHostKeyChecking=no"},
			[]string{"-v", "-o", "StrictHostKeyChecking=no", "me@10.0.0.5"}},
	}
	for _, c := range cases {
		got := buildSSHArgs(c.user, c.ip, c.port, c.id, c.extra)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestJoinCommand(t *testing.T) {
	got := joinCommand("ssh", []string{"-p", "2222", "root@10.0.0.5"})
	if got != "ssh -p 2222 root@10.0.0.5" {
		t.Errorf("plain: %q", got)
	}
	got = joinCommand("ssh", []string{"-i", "/path/with space/key", "root@10.0.0.5"})
	if got != `ssh -i '/path/with space/key' root@10.0.0.5` {
		t.Errorf("quoted: %q", got)
	}
}
