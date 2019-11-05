package mp4util

import (
	"net/http"
	"testing"
)

var urls = []string{
	"http://techslides.com/demos/samples/sample.mov",
	"http://techslides.com/demos/samples/sample.mp4",
}

func TestDuration(t *testing.T) {
	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatal("unexpected status code")
		}

		d, err := Duration(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("duration=%s", d)
	}
}
