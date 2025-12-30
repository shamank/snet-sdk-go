package model

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestServiceMetadata_GetMpeAddr(t *testing.T) {
	tests := []struct {
		name       string
		mpeAddress string
		want       common.Address
	}{
		{
			name:       "Valid address",
			mpeAddress: "0x1234567890123456789012345678901234567890",
			want:       common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		{
			name:       "Lowercase address",
			mpeAddress: "0xabcdef1234567890123456789012345678901234",
			want:       common.HexToAddress("0xabcdef1234567890123456789012345678901234"),
		},
		{
			name:       "Zero address",
			mpeAddress: "0x0000000000000000000000000000000000000000",
			want:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceMetadata{
				MPEAddress: tt.mpeAddress,
			}
			got := s.GetMpeAddr()
			if got != tt.want {
				t.Fatalf("GetMpeAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}
