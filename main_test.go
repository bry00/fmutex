package main

import (
	"fmt"
	"os"
	"path"
	"testing"
)

func temporaryCatalog(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "temp-*.dir")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("error removing temporary directory: %v", err)
		}
	})
	return tempDir
}

func TestIsEmptyStr(t *testing.T) {
	cases := [][]string{
		{"", "  ", " \t "},
		{"A", " a "},
	}

	for i, c := range cases {
		expected := i == 0
		for _, testCase := range c {
			if got := isEmptyStr(testCase); got != expected {
				t.Fatalf("wrong value of isEmpty(\"%s\") => %v instead of %v", testCase, got, expected)
			}
		}
	}
}

func TestIfEmptyStr(t *testing.T) {
	cases := []struct {
		s string
		r string
	}{
		{"", "abcd"},
		{"xyz", "xyz"},
		{"   ", "abcd"},
	}

	for _, c := range cases {
		expected := c.r
		if got := ifEmptyStr(c.s, "abcd"); got != expected {
			t.Fatalf("wrong value of ifEmpty(\"%s\", \"abcd\") => %v instead of \"%s\"", c.s, got, expected)
		}
	}
}

func lockName() string {
	lockFile := fmt.Sprintf("%s-mutex.lck", cmn.Id)
	return path.Join(cmn.Root, cmn.Id, lockFile)
}

func TestTest(t *testing.T) {
	cmn.Root = temporaryCatalog(t)
	cmn.Id = "test-test"
	doLock()
	expected := 0
	if got := doTest(); got != expected {
		t.Fatalf("wrong value of doTest() => %d instead of %d", got, expected)
	}
	doUnlock()
	expected = 1
	if got := doTest(); got != expected {
		t.Fatalf("wrong value of doTest() => %d instead of %d", got, expected)
	}
}

func TestLock(t *testing.T) {
	cmn.Root = temporaryCatalog(t)
	cmn.Id = "test-lock"
	defer doUnlock()
	doLock()
	expected := lockName()
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("wrong result of doLock(): %v", err)
	}
}

func TestUnlock(t *testing.T) {
	cmn.Root = temporaryCatalog(t)
	cmn.Id = "test-unlock"
	doLock()
	doUnlock()
	lockFile := lockName()
	if _, err := os.Stat(lockFile); err == nil {
		t.Fatalf("wrong result of doUnlock(): lock file still exists: %s", lockFile)
	}
}
