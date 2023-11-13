package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEmbedBinName(t *testing.T) {
	exe := "pikpakcli.exe"
	exe2 := ".exe"
	assert.Equal(t, "pikpakcli_embed.exe", GetEmbedBinName(exe))
	assert.Equal(t, "_embed.exe", GetEmbedBinName(exe2))

	bin := "pikpakcli"
	bin2 := ""
	assert.Equal(t, "pikpakcli_embed", GetEmbedBinName(bin))
	assert.Equal(t, "_embed", GetEmbedBinName(bin2))

}
