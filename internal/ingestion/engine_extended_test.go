package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsColon(t *testing.T) {
	tests := []struct {
		s       string
		hasColon bool
	}{
		{"host:993", true},
		{"imap.gmail.com", false},
		{":", true},
		{"", false},
		{"host:port:extra", true},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			assert.Equal(t, tt.hasColon, containsColon(tt.s), "containsColon(%q)", tt.s)
		})
	}
}

func TestLooksLikeNonDecimalIP_EdgeCases(t *testing.T) {
	tests := []struct {
		host    string
		isBadIP bool
	}{
		// Normal IPs
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"8.8.8.8", false},
		{"0.0.0.0", false}, // "0" alone as first octet is valid decimal
		{"0.0.0.1", false},

		// Octal bypasses
		{"0177.0.0.1", true},
		{"0251.0.0.1", true},
		{"01.0.0.1", true},

		// Hex bypasses
		{"0x7f.0.0.1", true},
		{"0X7F.0.0.1", true},
		{"0x0.0.0.1", true},

		// Integer form bypass
		{"2130706433", true},
		{"3232235521", true},
		{"0", true}, // single "0" is parseable as int

		// Non-bypass domains/hosts
		{"example.com", false},
		{"api.github.com", false},
		{"localhost", false},
		{"my-host.internal", false},

		// Edge cases
		{"", false},
		{"...", false},
		{"1.2.3.4.5", false}, // 5 parts, won't match 4-part or single-part
		{"10.0.0.01", true},  // "01" has leading zero but is valid octal... actually "01" could be octal but 1 is same in octal/decimal

		{"0", true}, // single "0" is parseable as int
		{"0x", false}, // not 4 parts, not a single int
		{"0xGG", false}, // not parseable as int
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := looksLikeNonDecimalIP(tt.host)
			assert.Equal(t, tt.isBadIP, got, "looksLikeNonDecimalIP(%q)", tt.host)
		})
	}
}

func TestNewEngine(t *testing.T) {
	type args struct {
		projectsRoot string
	}
	tests := []struct {
		name string
		args args
	}{
		{"empty root", args{""}},
		{"with path", args{"/tmp/test-projects"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(tt.args.projectsRoot, nil, nil, nil)
			assert.NotNil(t, eng)
			assert.Equal(t, tt.args.projectsRoot, eng.projectsRoot)
			assert.Nil(t, eng.metaRepo)
			assert.Nil(t, eng.db)
			assert.Nil(t, eng.nlpHandler)
			assert.NotNil(t, eng.tasks)
			assert.Empty(t, eng.tasks)
		})
	}
}

func TestNewEngine_WithDependencies(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	assert.NotNil(t, eng.tasks)
	assert.Empty(t, eng.tasks)

	// Map should be ready to use
	eng.tasks["test"] = nil
	assert.Contains(t, eng.tasks, "test")
}

func TestBlockSSRF_EmptyURL(t *testing.T) {
	err := blockSSRF("")
	assert.Error(t, err)
}

func TestBlockSSRF_InvalidURL(t *testing.T) {
	err := blockSSRF("://invalid")
	assert.Error(t, err)
}

func TestBlockSSRF_NoHost(t *testing.T) {
	err := blockSSRF("http:///path")
	assert.Error(t, err)
}

func TestBlockSSRF_0_0_0_0(t *testing.T) {
	err := blockSSRF("http://0.0.0.0:11434")
	assert.Error(t, err)
}

func TestBlockSSRF_InternalTLDs(t *testing.T) {
	assert.Error(t, blockSSRF("http://service.internal/api"))
	assert.Error(t, blockSSRF("http://dev.local:8080"))
	assert.Error(t, blockSSRF("http://host.arpa:8080"))
}

func TestBlockSSRF_IPv6Loopback(t *testing.T) {
	err := blockSSRF("http://[::1]:8080/api")
	assert.Error(t, err)

	err = blockSSRF("http://[0:0:0:0:0:0:0:1]:8080")
	assert.Error(t, err)
}

func TestBlockSSRF_NonDecimalIP(t *testing.T) {
	err := blockSSRF("http://0177.0.0.1:11434")
	assert.Error(t, err)

	err = blockSSRF("http://0x7f.0.0.1:11434")
	assert.Error(t, err)
}

func TestBlockSSRF_ValidExternal(t *testing.T) {
	err := blockSSRF("https://api.example.com/data")
	assert.NoError(t, err)

	err = blockSSRF("https://api.github.com/repos/owner/repo")
	assert.NoError(t, err)
}

func TestEngine_CloseExtended(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	err := eng.Close()
	assert.NoError(t, err)
}

func TestEngine_CloseMultiple(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	assert.NoError(t, eng.Close())
	assert.NoError(t, eng.Close()) // Closing twice should be safe
}

func TestVerifyChecksum_EmptyExpected(t *testing.T) {
	assert.False(t, VerifyChecksum([]byte("data"), ""))
}

func TestVerifyChecksum_ShortExpected(t *testing.T) {
	assert.False(t, VerifyChecksum([]byte("data"), "short"))
}

func TestBlockSSRF_PrivateIPs(t *testing.T) {
	tests := []struct {
		url string
	}{
		{"http://10.0.0.1/api"},
		{"http://172.16.0.1"},
		{"http://192.168.1.1"},
		{"http://169.254.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := blockSSRF(tt.url)
			assert.Error(t, err, "expected error for %q", tt.url)
		})
	}
}

func TestBlockSSRF_LocalhostVariantsExtended(t *testing.T) {
	assert.Error(t, blockSSRF("http://localhost:11434/api/tags"))
}

func TestBlockSSRF_ValidHTTPS(t *testing.T) {
	assert.NoError(t, blockSSRF("https://api.openai.com/v1/completions"))
	assert.NoError(t, blockSSRF("https://www.google.com"))
}
