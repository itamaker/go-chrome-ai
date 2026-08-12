package chrome

import (
	"reflect"
	"sort"
	"testing"
)

func TestSyncManagedFlags(t *testing.T) {
	cases := []struct {
		name         string
		input        map[string]any
		managed      []string
		selected     []string
		wantDisabled []string
		wantReverted []string
		wantList     []any
		wantNilBoth  bool
	}{
		{
			name:         "fresh local state, no browser key",
			input:        map[string]any{},
			managed:      []string{"foo"},
			selected:     []string{"foo"},
			wantDisabled: []string{"foo"},
			wantList:     []any{"foo@2"},
		},
		{
			name: "flag already disabled is left alone",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@2"},
				},
			},
			managed:     []string{"foo"},
			selected:    []string{"foo"},
			wantNilBoth: true,
			wantList:    []any{"foo@2"},
		},
		{
			name: "flag previously enabled is flipped to disabled",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@1", "bar"},
				},
			},
			managed:      []string{"foo"},
			selected:     []string{"foo"},
			wantDisabled: []string{"foo"},
			wantList:     []any{"bar", "foo@2"},
		},
		{
			name: "duplicates are coalesced",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@1", "foo@2"},
				},
			},
			managed:  []string{"foo"},
			selected: []string{"foo"},
			wantList: []any{"foo@2"},
		},
		{
			name: "multiple flags",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"keep"},
				},
			},
			managed:      []string{"foo", "bar"},
			selected:     []string{"foo", "bar"},
			wantDisabled: []string{"foo", "bar"},
			wantList:     []any{"keep", "foo@2", "bar@2"},
		},
		{
			name: "managed flag not selected is reverted",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@2", "other"},
				},
			},
			managed:      []string{"foo"},
			selected:     nil,
			wantReverted: []string{"foo"},
			wantList:     []any{"other"},
		},
		{
			name:        "managed flag not selected and not present is a no-op",
			input:       map[string]any{},
			managed:     []string{"foo"},
			selected:    nil,
			wantNilBoth: true,
			wantList:    []any{},
		},
		{
			name: "mixed: one applied, one reverted, unmanaged flag untouched",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@2", "bar@1", "unmanaged@1"},
				},
			},
			managed:      []string{"foo", "bar"},
			selected:     []string{"bar"},
			wantReverted: []string{"foo"},
			wantDisabled: []string{"bar"},
			wantList:     []any{"unmanaged@1", "bar@2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotDisabled, gotReverted := syncManagedFlags(tc.input, tc.managed, tc.selected)
			if tc.wantNilBoth {
				if gotDisabled != nil || gotReverted != nil {
					t.Fatalf("expected nil disabled/reverted, got disabled=%v reverted=%v", gotDisabled, gotReverted)
				}
			} else {
				if len(tc.wantDisabled) > 0 {
					sort.Strings(gotDisabled)
					want := append([]string(nil), tc.wantDisabled...)
					sort.Strings(want)
					if !reflect.DeepEqual(gotDisabled, want) {
						t.Fatalf("disabled list mismatch: got %v want %v", gotDisabled, want)
					}
				}
				if len(tc.wantReverted) > 0 {
					sort.Strings(gotReverted)
					want := append([]string(nil), tc.wantReverted...)
					sort.Strings(want)
					if !reflect.DeepEqual(gotReverted, want) {
						t.Fatalf("reverted list mismatch: got %v want %v", gotReverted, want)
					}
				}
			}
			browser, _ := tc.input["browser"].(map[string]any)
			if browser == nil {
				t.Fatalf("expected browser key to exist")
			}
			gotList, _ := browser["enabled_labs_experiments"].([]any)
			if len(gotList) == 0 && len(tc.wantList) == 0 {
				return
			}
			if !reflect.DeepEqual(gotList, tc.wantList) {
				t.Fatalf("list mismatch: got %v want %v", gotList, tc.wantList)
			}
		})
	}
}
