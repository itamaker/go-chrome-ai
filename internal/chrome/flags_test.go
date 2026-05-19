package chrome

import (
	"reflect"
	"sort"
	"testing"
)

func TestSetFlagsDisabled(t *testing.T) {
	cases := []struct {
		name      string
		input     map[string]any
		flags     []string
		want      []string
		wantList  []any
		wantNil   bool
	}{
		{
			name:  "fresh local state, no browser key",
			input: map[string]any{},
			flags: []string{"foo"},
			want:  []string{"foo"},
			wantList: []any{"foo@2"},
		},
		{
			name: "flag already disabled is left alone",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@2"},
				},
			},
			flags:    []string{"foo"},
			wantNil:  true,
			wantList: []any{"foo@2"},
		},
		{
			name: "flag previously enabled is flipped to disabled",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@1", "bar"},
				},
			},
			flags:    []string{"foo"},
			want:     []string{"foo"},
			wantList: []any{"bar", "foo@2"},
		},
		{
			name: "duplicates are coalesced",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@1", "foo@2"},
				},
			},
			flags:    []string{"foo"},
			wantList: []any{"foo@2"},
		},
		{
			name: "multiple flags",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"keep"},
				},
			},
			flags:    []string{"foo", "bar"},
			want:     []string{"foo", "bar"},
			wantList: []any{"keep", "foo@2", "bar@2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := setFlagsDisabled(tc.input, tc.flags)
			if tc.wantNil && got != nil {
				t.Fatalf("expected nil changed list, got %v", got)
			}
			if !tc.wantNil && len(tc.want) > 0 {
				sort.Strings(got)
				want := append([]string(nil), tc.want...)
				sort.Strings(want)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("changed list mismatch: got %v want %v", got, want)
				}
			}
			browser, _ := tc.input["browser"].(map[string]any)
			if browser == nil {
				t.Fatalf("expected browser key to exist")
			}
			gotList, _ := browser["enabled_labs_experiments"].([]any)
			if !reflect.DeepEqual(gotList, tc.wantList) {
				t.Fatalf("list mismatch: got %v want %v", gotList, tc.wantList)
			}
		})
	}
}
