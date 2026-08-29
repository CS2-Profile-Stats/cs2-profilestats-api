// BSD 2-Clause License
// Copyright (c) 2020, Emily Hudson
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// stolen from repo: https://github.com/emily33901/go-csfriendcode

package steam

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"math/bits"
)

const alnum = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// b32 is some kind of base32 that is used
// to encode the value that is created from the initial
// function. For some reason that I cant figure out
// it creates a different result to using go's base32 function
// even though they should basically do exactly the same thing

func b32(input uint64) string {
	result := []byte{}

	// big endian the number
	input = bits.ReverseBytes64(input)

	for i := range 13 {
		if i == 4 || i == 9 {
			result = append(result, '-')
		}
		result = append(result, alnum[input&0x1F])
		input >>= 5
	}

	return string(result)
}

func hashSteamID(id uint64) (uint32, error) {
	accountID := uint32(id)
	strangeSteamID := uint64(accountID) | 0x4353474F00000000

	steamIDBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(steamIDBytes, strangeSteamID)

	h := md5.New()
	n, err := h.Write(steamIDBytes)
	if err != nil {
		return 0, err
	}

	if n != 8 {
		return 0, fmt.Errorf("Couldnt hash steamid")
	}

	hashSteamID := h.Sum(nil)
	return binary.LittleEndian.Uint32(hashSteamID), nil
}

// makeU64 takes a high and low uint32
// and makes a uint64 out of them
func makeU64(hi uint32, lo uint32) uint64 {
	return uint64((uint64(hi) << 32) | uint64(lo))
}

func friendCode(id uint64) string {
	h, err := hashSteamID(id)
	if err != nil {
		return ""
	}

	r := uint64(0)
	for i := range 8 {
		idNibble := byte(id & 0xF)
		id >>= 4

		hashNibble := (h >> i) & 1

		a := uint32(r<<4) | uint32(idNibble)

		r = makeU64(uint32(r>>28), a)
		r = makeU64(uint32(r>>31), a<<1|hashNibble)
	}

	return b32(r)
}

// gets a friend code based on a provided steamid64
func steamIdToFriendCode(id uint64) string {
	fc := friendCode(id)
	if fc[:5] == "AAAA-" {
		fc = fc[5:]
	}
	return fc
}
