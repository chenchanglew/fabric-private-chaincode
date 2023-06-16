package tlecore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetaState(t *testing.T) {
	ms := &Tlestate{}

	err := ms.PutMeta("example1", "key1", []byte("metadata1"))
	require.NoError(t, err)
	err = ms.PutMeta("example2", "key2", []byte("metadata2"))
	require.NoError(t, err)

	meta, err := ms.GetMeta("example1", "key1")
	require.NoError(t, err)
	require.Equal(t, meta, []byte("metadata1"))

	meta, err = ms.GetMeta("example2", "key2")
	require.NoError(t, err)
	require.Equal(t, meta, []byte("metadata2"))

	meta, err = ms.GetMeta("example", "key1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "namespace not found")
	meta, err = ms.GetMeta("example2", "key1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "key not found")
}
