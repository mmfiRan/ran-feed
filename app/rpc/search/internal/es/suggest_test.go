package es

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSuggestOptions(t *testing.T) {
	body := `{
		"suggest": {
			"suggest": [
				{
					"text": "露",
					"options": [
						{"text": "露营装备测评"},
						{"text": "露天电影院"}
					]
				}
			]
		}
	}`

	texts, err := parseSuggestOptions(strings.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, []string{"露营装备测评", "露天电影院"}, texts)
}

func TestParseSuggestOptionsEmpty(t *testing.T) {
	// 无命中 options 为空
	texts, err := parseSuggestOptions(strings.NewReader(`{"suggest":{"suggest":[{"text":"zzz","options":[]}]}}`))
	require.NoError(t, err)
	assert.Empty(t, texts)

	// 完全没有 suggest 块
	texts, err = parseSuggestOptions(strings.NewReader(`{}`))
	require.NoError(t, err)
	assert.Empty(t, texts)
}

func TestParseSuggestOptionsBadJSON(t *testing.T) {
	_, err := parseSuggestOptions(strings.NewReader(`{not json`))
	assert.Error(t, err)
}
