package requests

import (
	"testing"
)

func TestNewRequest(t *testing.T) {
	client, err := GetHttpClient("")
	if err != nil {
		t.Fatal(err)
	}

	req, err := client.NewRequest("GET", "http://google.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	uaHeader := req.Header.Get("User-Agent")
	if uaHeader != "Harpoon-Client/0_1" {
		t.Errorf("Unexpected \"User-Agent\" header %s", uaHeader)
	}
}
