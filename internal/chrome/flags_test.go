package chrome

import (
	"reflect"
	"sort"
	"testing"
)

// normalizeStrings returns a sorted copy of s, treating nil and empty as
// equivalent, so tests don't have to care whether a code path returns a nil
// slice or an empty one, or in what order names were appended.
func normalizeStrings(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// normalizeAnyList returns l with nil normalized to an empty, non-nil
// slice, so tests don't have to care whether "no items" comes back as nil
// or []any{}.
func normalizeAnyList(l []any) []any {
	if len(l) == 0 {
		return []any{}
	}
	return l
}

func TestSyncManagedFlags(t *testing.T) {
	cases := []struct {
		name         string
		input        map[string]any
		managed      []string
		selected     []string
		wantDisabled []string
		wantReverted []string
		// wantNoBrowserKey asserts localState["browser"] was never created
		// (true no-op path). Mutually exclusive with wantList.
		wantNoBrowserKey bool
		wantList         []any
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
			managed:  []string{"foo"},
			selected: []string{"foo"},
			wantList: []any{"foo@2"},
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
			name: "duplicates in existing state are coalesced",
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
			name:         "duplicate names in selected are coalesced into one entry",
			input:        map[string]any{},
			managed:      []string{"foo"},
			selected:     []string{"foo", "foo", "foo"},
			wantDisabled: []string{"foo"},
			wantList:     []any{"foo@2"},
		},
		{
			name:         "name in selected that is not in managed is ignored",
			input:        map[string]any{},
			managed:      []string{"foo"},
			selected:     []string{"foo", "not-a-real-flag"},
			wantDisabled: []string{"foo"},
			wantList:     []any{"foo@2"},
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
			name:             "managed flag not selected and not present is a no-op, and creates no browser key",
			input:            map[string]any{},
			managed:          []string{"foo"},
			selected:         nil,
			wantNoBrowserKey: true,
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
		{
			name: "non-string entries in the existing list survive a rewrite untouched",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{"foo@1", float64(999), "bar", true, nil},
				},
			},
			managed:      []string{"foo"},
			selected:     []string{"foo"},
			wantDisabled: []string{"foo"},
			wantList:     []any{float64(999), "bar", true, nil, "foo@2"},
		},
		{
			name: "non-string entries are untouched when nothing else changes",
			input: map[string]any{
				"browser": map[string]any{
					"enabled_labs_experiments": []any{float64(1), "unmanaged"},
				},
			},
			managed:  []string{"foo"},
			selected: nil,
			wantList: []any{float64(1), "unmanaged"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotDisabled, gotReverted := syncManagedFlags(tc.input, tc.managed, tc.selected)

			if !reflect.DeepEqual(normalizeStrings(gotDisabled), normalizeStrings(tc.wantDisabled)) {
				t.Fatalf("disabled list mismatch: got %v want %v", gotDisabled, tc.wantDisabled)
			}
			if !reflect.DeepEqual(normalizeStrings(gotReverted), normalizeStrings(tc.wantReverted)) {
				t.Fatalf("reverted list mismatch: got %v want %v", gotReverted, tc.wantReverted)
			}

			if tc.wantNoBrowserKey {
				if v, exists := tc.input["browser"]; exists {
					t.Fatalf("expected no browser key to be created, got %v", v)
				}
				return
			}

			browser, ok := tc.input["browser"].(map[string]any)
			if !ok {
				t.Fatalf("expected browser key to exist as a map, got %v", tc.input["browser"])
			}
			gotList, _ := browser["enabled_labs_experiments"].([]any)
			if !reflect.DeepEqual(normalizeAnyList(gotList), normalizeAnyList(tc.wantList)) {
				t.Fatalf("list mismatch: got %v want %v", gotList, tc.wantList)
			}
		})
	}
}
