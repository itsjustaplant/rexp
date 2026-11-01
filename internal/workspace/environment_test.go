package workspace

import (
	"reflect"
	"runtime"
	"testing"
)

func TestEnvironmentPrecedenceWithoutMutation(t *testing.T) {
	project := map[string]string{"LEVEL": "project", "PROJECT": "yes"}
	tab := map[string]string{"LEVEL": "tab", "TAB": "yes"}
	pane := map[string]string{"LEVEL": "pane", "EMPTY": ""}
	merged := MergeEnvironment(project, tab, pane)
	want := map[string]string{"LEVEL": "pane", "PROJECT": "yes", "TAB": "yes", "EMPTY": ""}
	if !reflect.DeepEqual(merged, want) || project["LEVEL"] != "project" || tab["LEVEL"] != "tab" {
		t.Fatalf("merge: %v", merged)
	}
	base := []string{"INHERITED=yes", "LEVEL=old", "EMPTY=old", "=C:=C:\\project"}
	before := append([]string(nil), base...)
	result := OverlayEnvironment(base, merged)
	if !reflect.DeepEqual(base, before) {
		t.Fatal("modified inherited environment")
	}
	if !reflect.DeepEqual(result, []string{"INHERITED=yes", "=C:=C:\\project", "EMPTY=", "LEVEL=pane", "PROJECT=yes", "TAB=yes"}) {
		t.Fatalf("overlay: %v", result)
	}
}

func TestNativeEnvironmentCaseRules(t *testing.T) {
	merged := MergeEnvironment(map[string]string{"Path": "first"}, map[string]string{"PATH": "second"})
	if runtime.GOOS == "windows" {
		if len(merged) != 1 || merged["PATH"] != "second" {
			t.Fatalf("Windows case merge: %v", merged)
		}
		if err := ValidateEnvironment(map[string]string{"Path": "a", "PATH": "b"}, "env"); err == nil {
			t.Fatal("accepted duplicate Windows names")
		}
	} else if len(merged) != 2 {
		t.Fatalf("Unix case merge: %v", merged)
	}
}
