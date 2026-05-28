package utils

import "testing"

func TestDetectSource(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=abc":     "yt",
		"https://youtu.be/abc":                    "yt",
		"https://github.com/foo/bar":              "gh",
		"https://arxiv.org/abs/1706.03762":        "arxiv",
		"https://huggingface.co/tiiuae/falcon-7b": "hf",
		"https://twitter.com/someone":             "x",
		"https://x.com/someone":                   "x",
		"https://www.reddit.com/r/golang":         "reddit",
		"https://go.dev/blog/go1.22":              "web",
		"not-a-url":                               "",
	}
	for in, want := range cases {
		if got := DetectSource(in); got != want {
			t.Errorf("DetectSource(%q) = %q, want %q", in, got, want)
		}
	}
}
