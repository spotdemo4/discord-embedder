package config

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/caarlos0/env/v11"
)

func TestKeyUnmarshalText(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)

	tests := []struct {
		name    string
		text    string
		want    Key
		wantErr bool
	}{
		{name: "valid", text: base64.StdEncoding.EncodeToString(key), want: key},
		{name: "trailing newline", text: base64.StdEncoding.EncodeToString(key) + "\n", want: key},
		{name: "not base64", text: "not base64!", wantErr: true},
		{name: "too short", text: base64.StdEncoding.EncodeToString(key[:16]), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Key
			err := got.UnmarshalText([]byte(tt.text))
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("UnmarshalText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyFromEnv(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	t.Setenv("COOKIES_KEY", base64.StdEncoding.EncodeToString(key))

	var cfg struct {
		CookiesKey Key `env:"COOKIES_KEY"`
	}
	if err := env.Parse(&cfg); err != nil {
		t.Fatalf("env.Parse() error = %v", err)
	}
	if !bytes.Equal(cfg.CookiesKey, key) {
		t.Errorf("CookiesKey = %v, want %v", cfg.CookiesKey, key)
	}
}
