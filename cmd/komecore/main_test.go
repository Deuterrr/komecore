package main

import (
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      opts
		wantError bool
	}{
		{
			name: "empty args runs server",
			args: []string{},
			want: opts{},
		},
		{
			name: "serve subcommand runs server",
			args: []string{"serve"},
			want: opts{},
		},
		{
			name: "version subcommand",
			args: []string{"version"},
			want: opts{showVersion: true},
		},
		{
			name: "version shorthand flag",
			args: []string{"-v"},
			want: opts{showVersion: true},
		},
		{
			name: "migrate default to up",
			args: []string{"migrate"},
			want: opts{migrate: true},
		},
		{
			name: "migrate up",
			args: []string{"migrate", "up"},
			want: opts{migrate: true},
		},
		{
			name: "migrate rollback",
			args: []string{"migrate", "rollback"},
			want: opts{rollback: true},
		},
		{
			name: "migrate down",
			args: []string{"migrate", "down"},
			want: opts{migrateDown: true},
		},
		{
			name: "migrate version",
			args: []string{"migrate", "version"},
			want: opts{migrateVersion: true},
		},
		{
			name: "migrate up with target flag",
			args: []string{"migrate", "up", "--target=supabase"},
			want: opts{migrate: true, target: "supabase"},
		},
		{
			name: "migrate target before action",
			args: []string{"migrate", "--target", "sqlserver", "rollback"},
			want: opts{rollback: true, target: "sqlserver"},
		},
		{
			name:      "migrate unknown action errors",
			args:      []string{"migrate", "unknown"},
			wantError: true,
		},
		{
			name: "seed base",
			args: []string{"seed"},
			want: opts{seed: true},
		},
		{
			name: "seed dev positional",
			args: []string{"seed", "dev"},
			want: opts{seed: true, seedDev: true},
		},
		{
			name: "seed dev flag with force",
			args: []string{"seed", "--dev", "--force"},
			want: opts{seed: true, seedDev: true, force: true},
		},
		{
			name: "seed dev positional with force and target",
			args: []string{"seed", "dev", "--force", "--target=supabase"},
			want: opts{seed: true, seedDev: true, force: true, target: "supabase"},
		},
		{
			name:      "seed unknown action errors",
			args:      []string{"seed", "invalid"},
			wantError: true,
		},
		{
			name:      "unknown command errors",
			args:      []string{"unknown"},
			wantError: true,
		},
		{
			name:      "legacy flag errors",
			args:      []string{"--migrate"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArgs(tc.args)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error for args %v, got nil", tc.args)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for args %v: %v", tc.args, err)
			}

			if got != tc.want {
				t.Errorf("parseArgs(%v) =\n%+v\nwant:\n%+v", tc.args, got, tc.want)
			}
		})
	}
}
