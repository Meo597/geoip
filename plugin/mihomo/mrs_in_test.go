package mihomo

import (
	"bytes"
	"encoding/binary"
	"github.com/klauspost/compress/zstd"
	"github.com/xtls/geoip/lib"
	"math"
	"slices"
	"testing"
)

func TestMRSExtraData(t *testing.T) {
	original := lib.NewEntry("test")
	if err := original.AddPrefix("192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	ranges, err := original.MarshalIPRange()
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	if err := (&MRSOut{}).convertToMrs(ranges, &compressed); err != nil {
		t.Fatal(err)
	}
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := decoder.DecodeAll(compressed.Bytes(), nil)
	decoder.Close()
	if err != nil {
		t.Fatal(err)
	}
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	for _, tc := range []struct {
		name      string
		length    int64
		extra     []byte
		truncated bool
		wantError bool
	}{
		{name: "none"},
		{name: "skip reserved data", length: 3, extra: []byte{1, 2, 3}},
		{name: "huge length", length: math.MaxInt64, truncated: true, wantError: true},
		{name: "truncated extra", length: 4, extra: []byte{1, 2}, truncated: true, wantError: true},
		{name: "negative length", length: -1, truncated: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := append([]byte(nil), raw[:21]...)
			binary.BigEndian.PutUint64(payload[13:21], uint64(tc.length))
			payload = append(payload, tc.extra...)
			if !tc.truncated {
				payload = append(payload, raw[21:]...)
			}
			entry := lib.NewEntry("test")
			err := (&MRSIn{}).parseMRS(encoder.EncodeAll(payload, nil), entry)
			if (err != nil) != tc.wantError {
				t.Fatalf("parseMRS error = %v", err)
			}
			if !tc.wantError {
				got, err := entry.MarshalText()
				if err != nil || !slices.Equal(got, []string{"192.0.2.0/24"}) {
					t.Fatalf("rules changed after skipping extra data: %v, %v", got, err)
				}
			}
		})
	}
}
