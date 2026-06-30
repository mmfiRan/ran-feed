package es

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHitFirstHighlight(t *testing.T) {
	h := Hit{Highlight: map[string][]string{
		"title": {"露<em>营</em>装备", "第二片段"},
	}}

	assert.Equal(t, "露<em>营</em>装备", h.FirstHighlight("title"))
	assert.Empty(t, h.FirstHighlight("description"))

	empty := Hit{}
	assert.Empty(t, empty.FirstHighlight("title"))
}
