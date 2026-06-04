package utils

import (
	"crypto/rand"
	"math/big"
)

const codeAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const inviteCodeLength = 7 // 7 characters = 62^7 = ~3.5 trillion combos

// Generate random code using alphanumerics i.e 62 possibilities for each position.
// Total combos 62^codelength
func generateRandomCode(codeLength int) string {
	// Birthday Pardox
	// If you generate n codes, the chance that at least two are the same is roughly:
	// 1 - e^(-n² / (2 × totalCodeAlphabets^codeLength))
	code := make([]byte, codeLength)
	max := big.NewInt(int64(len(codeAlphabet)))

	for i := range code {
		num, _ := rand.Int(rand.Reader, max)
		code[i] = codeAlphabet[num.Int64()]
	}
	return string(code)
}

// Generate invite code
func GenerateInviteCode() string {
	return generateRandomCode(inviteCodeLength)
}
