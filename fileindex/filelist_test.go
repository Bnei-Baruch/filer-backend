package fileindex

import (
	"bufio"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	f := bufio.NewReader(strings.NewReader(FilesBadJson))
	l, err := Load(f, nil)
	if err == nil {
		t.Errorf("Expected: JSON input error")
	}
	if len(l) > 0 {
		t.Errorf("Length = %d, expected lenght = 0", len(l))
	}

	for i, files := range Files {
		expect := FilesRecords[i]
		f := bufio.NewReader(strings.NewReader(files))
		l, err := Load(f, nil)
		if err != nil {
			t.Errorf("Error: %v", err)
		} else if len(l) != expect {
			t.Errorf("Load records = %d, expected %d", len(l), expect)
		}
	}
}
