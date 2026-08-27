package main

import "testing"

func TestAddressResolution(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		port    string
		want    string
		wantErr bool
	}{
		{"default", nil, "", defaultAddress, false}, {"port", nil, "19111", "127.0.0.1:19111", false},
		{"flag wins", []string{"-addr=127.0.0.1:19222"}, "19111", "127.0.0.1:19222", false},
		{"reject public", []string{"-addr=0.0.0.0:19081"}, "", "", true}, {"reject invalid port", nil, "abc", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration, err := parseConfig(test.args, func(string) string { return test.port })
			if (err != nil) != test.wantErr {
				t.Fatalf("err=%v", err)
			}
			if err == nil && configuration.Address != test.want {
				t.Fatalf("want %s got %s", test.want, configuration.Address)
			}
		})
	}
}
