package rest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/stretchr/testify/require"
)

func TestValidateVirtioFsShare(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, nil, 0o600))

	valid := []define.VirtioFsShare{
		{SharedDir: dir},
		{SharedDir: dir, ReadOnly: true},
		{Shares: map[string]define.SharedDir{"a": {SharedDir: dir}, "b": {SharedDir: dir, ReadOnly: true}}},
		{Shares: map[string]define.SharedDir{}},
	}
	for _, req := range valid {
		require.NoError(t, validateVirtioFsShare(req), "%+v", req)
	}

	invalid := []define.VirtioFsShare{
		{},
		{SharedDir: dir, Shares: map[string]define.SharedDir{}},
		{SharedDir: "relative"},
		{SharedDir: filepath.Join(dir, "missing")},
		{SharedDir: file},
		{Shares: map[string]define.SharedDir{"": {SharedDir: dir}}},
		{Shares: map[string]define.SharedDir{"..": {SharedDir: dir}}},
		{Shares: map[string]define.SharedDir{"a/b": {SharedDir: dir}}},
		{Shares: map[string]define.SharedDir{"a": {SharedDir: "relative"}}},
	}
	for _, req := range invalid {
		require.Error(t, validateVirtioFsShare(req), "%+v", req)
	}
}
