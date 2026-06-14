package itunes

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "itunes" {
		t.Errorf("Scheme = %q, want itunes", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "itunes" {
		t.Errorf("Identity.Binary = %q, want itunes", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("12345")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "result" {
		t.Errorf("type = %q, want result", typ)
	}
	if id != "12345" {
		t.Errorf("id = %q, want 12345", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("result", "12345")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	want := "https://" + Host + "/lookup?id=12345"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "foo")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}
