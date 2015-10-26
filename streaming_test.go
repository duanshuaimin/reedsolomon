/**
 * Unit tests for ReedSolomon Streaming API
 *
 * Copyright 2015, Klaus Post
 */

package reedsolomon

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

func TestStreamEncoding(t *testing.T) {
	perShard := 10 << 20
	if testing.Short() {
		perShard = 50000
	}
	r, err := NewStream(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	rand.Seed(0)
	input := randomBytes(10, perShard)
	data := toBuffers(input)
	par := emptyBuffers(3)

	err = r.Encode(toReaders(data), toWriters(par))
	if err != nil {
		t.Fatal(err)
	}
	// Reset Data
	data = toBuffers(input)

	all := append(toReaders(data), toReaders(par)...)
	ok, err := r.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	err = r.Encode(toReaders(emptyBuffers(1)), toWriters(emptyBuffers(1)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Encode(toReaders(emptyBuffers(10)), toWriters(emptyBuffers(1)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Encode(toReaders(emptyBuffers(10)), toWriters(emptyBuffers(3)))
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}

	badShards := emptyBuffers(10)
	badShards[0] = randomBuffer(123)
	err = r.Encode(toReaders(badShards), toWriters(emptyBuffers(3)))
	if err != ErrShardSize {
		t.Errorf("expected %v, got %v", ErrShardSize, err)
	}
}

func TestStreamEncodingConcurrent(t *testing.T) {
	perShard := 10 << 20
	if testing.Short() {
		perShard = 50000
	}
	r, err := NewStreamC(10, 3, true, true)
	if err != nil {
		t.Fatal(err)
	}
	rand.Seed(0)
	input := randomBytes(10, perShard)
	data := toBuffers(input)
	par := emptyBuffers(3)

	err = r.Encode(toReaders(data), toWriters(par))
	if err != nil {
		t.Fatal(err)
	}
	// Reset Data
	data = toBuffers(input)

	all := append(toReaders(data), toReaders(par)...)
	ok, err := r.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	err = r.Encode(toReaders(emptyBuffers(1)), toWriters(emptyBuffers(1)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Encode(toReaders(emptyBuffers(10)), toWriters(emptyBuffers(1)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Encode(toReaders(emptyBuffers(10)), toWriters(emptyBuffers(3)))
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}

	badShards := emptyBuffers(10)
	badShards[0] = randomBuffer(123)
	err = r.Encode(toReaders(badShards), toWriters(emptyBuffers(3)))
	if err != ErrShardSize {
		t.Errorf("expected %v, got %v", ErrShardSize, err)
	}
}

func randomBuffer(length int) *bytes.Buffer {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte(rand.Int() & 0xff)
	}
	return bytes.NewBuffer(b)
}

func randomBytes(n, length int) [][]byte {
	bufs := make([][]byte, n)
	for j := range bufs {
		bufs[j] = make([]byte, length)
		for i := range bufs[j] {
			bufs[j][i] = byte(rand.Int() & 0xff)
		}
	}
	return bufs
}

func toBuffers(in [][]byte) []*bytes.Buffer {
	out := make([]*bytes.Buffer, len(in))
	for i := range in {
		out[i] = bytes.NewBuffer(in[i])
	}
	return out
}

func toReaders(in []*bytes.Buffer) []io.Reader {
	out := make([]io.Reader, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

func toWriters(in []*bytes.Buffer) []io.Writer {
	out := make([]io.Writer, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

func nilWriters(n int) []io.Writer {
	out := make([]io.Writer, n)
	for i := range out {
		out[i] = nil
	}
	return out
}

func emptyBuffers(n int) []*bytes.Buffer {
	b := make([]*bytes.Buffer, n)
	for i := range b {
		b[i] = &bytes.Buffer{}
	}
	return b
}

func toBytes(in []*bytes.Buffer) [][]byte {
	b := make([][]byte, len(in))
	for i := range in {
		b[i] = in[i].Bytes()
	}
	return b
}

func TestStreamReconstruct(t *testing.T) {
	perShard := 10 << 20
	if testing.Short() {
		perShard = 50000
	}
	r, err := NewStream(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	rand.Seed(0)
	shards := randomBytes(10, perShard)
	parb := emptyBuffers(3)

	err = r.Encode(toReaders(toBuffers(shards)), toWriters(parb))
	if err != nil {
		t.Fatal(err)
	}

	parity := toBytes(parb)

	all := append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)
	fill := make([]io.Writer, 13)

	// Reconstruct with all shards present, all fill nil
	err = r.Reconstruct(all, fill)
	if err != nil {
		t.Fatal(err)
	}

	all = append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)

	// Reconstruct with 10 shards present
	all[0] = nil
	fill[0] = emptyBuffers(1)[0]
	all[7] = nil
	fill[7] = emptyBuffers(1)[0]
	all[11] = nil
	fill[11] = emptyBuffers(1)[0]

	err = r.Reconstruct(all, fill)
	if err != nil {
		t.Fatal(err)
	}

	shards[0] = fill[0].(*bytes.Buffer).Bytes()
	shards[7] = fill[7].(*bytes.Buffer).Bytes()
	parity[1] = fill[11].(*bytes.Buffer).Bytes()

	all = append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)

	ok, err := r.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	all = append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)

	// Reconstruct with 9 shards present (should fail)
	all[0] = nil
	fill[0] = emptyBuffers(1)[0]
	all[4] = nil
	fill[4] = emptyBuffers(1)[0]
	all[7] = nil
	fill[7] = emptyBuffers(1)[0]
	all[11] = nil
	fill[11] = emptyBuffers(1)[0]

	err = r.Reconstruct(all, fill)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	err = r.Reconstruct(toReaders(emptyBuffers(3)), toWriters(emptyBuffers(3)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Reconstruct(toReaders(emptyBuffers(13)), toWriters(emptyBuffers(3)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Reconstruct(toReaders(emptyBuffers(13)), toWriters(emptyBuffers(13)))
	if err != ErrReconstructMismatch {
		t.Errorf("expected %v, got %v", ErrReconstructMismatch, err)
	}
	err = r.Reconstruct(toReaders(emptyBuffers(13)), nilWriters(13))
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}
}

func TestStreamVerify(t *testing.T) {
	perShard := 33333
	r, err := NewStream(10, 4)
	if err != nil {
		t.Fatal(err)
	}
	shards := randomBytes(10, perShard)
	parb := emptyBuffers(4)

	err = r.Encode(toReaders(toBuffers(shards)), toWriters(parb))
	if err != nil {
		t.Fatal(err)
	}
	parity := toBytes(parb)
	all := append(toReaders(toBuffers(shards)), toReaders(parb)...)
	ok, err := r.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	// Put in random data. Verification should fail
	fillRandom(parity[0])
	all = append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)
	ok, err = r.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("Verification did not fail")
	}
	// Re-encode
	err = r.Encode(toReaders(toBuffers(shards)), toWriters(parb))
	if err != nil {
		t.Fatal(err)
	}
	// Fill a data segment with random data
	fillRandom(shards[0])
	all = append(toReaders(toBuffers(shards)), toReaders(parb)...)
	ok, err = r.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("Verification did not fail")
	}

	_, err = r.Verify(toReaders(emptyBuffers(10)))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	_, err = r.Verify(toReaders(emptyBuffers(14)))
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}
}

func TestStreamOneEncode(t *testing.T) {
	codec, err := NewStream(5, 5)
	if err != nil {
		t.Fatal(err)
	}
	shards := [][]byte{
		{0, 1},
		{4, 5},
		{2, 3},
		{6, 7},
		{8, 9},
	}
	parb := emptyBuffers(5)
	codec.Encode(toReaders(toBuffers(shards)), toWriters(parb))
	parity := toBytes(parb)
	if parity[0][0] != 12 || parity[0][1] != 13 {
		t.Fatal("shard 5 mismatch")
	}
	if parity[1][0] != 10 || parity[1][1] != 11 {
		t.Fatal("shard 6 mismatch")
	}
	if parity[2][0] != 14 || parity[2][1] != 15 {
		t.Fatal("shard 7 mismatch")
	}
	if parity[3][0] != 90 || parity[3][1] != 91 {
		t.Fatal("shard 8 mismatch")
	}
	if parity[4][0] != 94 || parity[4][1] != 95 {
		t.Fatal("shard 9 mismatch")
	}

	all := append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)
	ok, err := codec.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("did not verify")
	}
	shards[3][0]++
	all = append(toReaders(toBuffers(shards)), toReaders(toBuffers(parity))...)
	ok, err = codec.Verify(all)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("verify did not fail as expected")
	}

}

// FIXME
func benchmarkStreamEncode(b *testing.B, dataShards, parityShards, shardSize int) {
	r, err := New(dataShards, parityShards)
	if err != nil {
		b.Fatal(err)
	}
	shards := make([][]byte, dataShards+parityShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}

	b.SetBytes(int64(shardSize * dataShards))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Encode(shards)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamEncode10x2x10000(b *testing.B) {
	benchmarkStreamEncode(b, 10, 2, 10000)
}

func BenchmarkStreamEncode100x20x10000(b *testing.B) {
	benchmarkStreamEncode(b, 100, 20, 10000)
}

func BenchmarkStreamEncode17x3x1M(b *testing.B) {
	benchmarkStreamEncode(b, 17, 3, 1024*1024)
}

// Benchmark 10 data shards and 4 parity shards with 16MB each.
func BenchmarkStreamEncode10x4x16M(b *testing.B) {
	benchmarkStreamEncode(b, 10, 4, 16*1024*1024)
}

// Benchmark 5 data shards and 2 parity shards with 1MB each.
func BenchmarkStreamEncode5x2x1M(b *testing.B) {
	benchmarkStreamEncode(b, 5, 2, 1024*1024)
}

// Benchmark 1 data shards and 2 parity shards with 1MB each.
func BenchmarkStreamEncode10x2x1M(b *testing.B) {
	benchmarkStreamEncode(b, 10, 2, 1024*1024)
}

// Benchmark 10 data shards and 4 parity shards with 1MB each.
func BenchmarkStreamEncode10x4x1M(b *testing.B) {
	benchmarkStreamEncode(b, 10, 4, 1024*1024)
}

// Benchmark 50 data shards and 20 parity shards with 1MB each.
func BenchmarkStreamEncode50x20x1M(b *testing.B) {
	benchmarkStreamEncode(b, 50, 20, 1024*1024)
}

// Benchmark 17 data shards and 3 parity shards with 16MB each.
func BenchmarkStreamEncode17x3x16M(b *testing.B) {
	benchmarkStreamEncode(b, 17, 3, 16*1024*1024)
}

// FIXME
func benchmarkStreamVerify(b *testing.B, dataShards, parityShards, shardSize int) {
	r, err := New(dataShards, parityShards)
	if err != nil {
		b.Fatal(err)
	}
	shards := make([][]byte, parityShards+dataShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}
	err = r.Encode(shards)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(shardSize * dataShards))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = r.Verify(shards)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark 10 data slices with 2 parity slices holding 10000 bytes each
func BenchmarkStreamVerify10x2x10000(b *testing.B) {
	benchmarkStreamVerify(b, 10, 2, 10000)
}

// Benchmark 50 data slices with 5 parity slices holding 100000 bytes each
func BenchmarkStreamVerify50x5x50000(b *testing.B) {
	benchmarkStreamVerify(b, 50, 5, 100000)
}

// Benchmark 10 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkStreamVerify10x2x1M(b *testing.B) {
	benchmarkStreamVerify(b, 10, 2, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkStreamVerify5x2x1M(b *testing.B) {
	benchmarkStreamVerify(b, 5, 2, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 1MB bytes each
func BenchmarkStreamVerify10x4x1M(b *testing.B) {
	benchmarkStreamVerify(b, 10, 4, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkStreamVerify50x20x1M(b *testing.B) {
	benchmarkStreamVerify(b, 50, 20, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 16MB bytes each
func BenchmarkStreamVerify10x4x16M(b *testing.B) {
	benchmarkStreamVerify(b, 10, 4, 16*1024*1024)
}

// Simple example of how to use all functions of the Encoder.
// Note that all error checks have been removed to keep it short.
// FIXME: Actually use NewStream
func ExampleStreamEncoder() {
	// Create some sample data
	var data = make([]byte, 250000)
	fillRandom(data)

	// Create an encoder with 17 data and 3 parity slices.
	enc, _ := New(17, 3)

	// Split the data into shards
	shards, _ := enc.Split(data)

	// Encode the parity set
	_ = enc.Encode(shards)

	// Verify the parity set
	ok, _ := enc.Verify(shards)
	if ok {
		fmt.Println("ok")
	}

	// Delete two shards
	shards[10], shards[11] = nil, nil

	// Reconstruct the shards
	_ = enc.Reconstruct(shards)

	// Verify the data set
	ok, _ = enc.Verify(shards)
	if ok {
		fmt.Println("ok")
	}
	// Output: ok
	// ok
}

func TestStreamSplitJoin(t *testing.T) {
	var data = make([]byte, 250000)
	rand.Seed(0)
	fillRandom(data)

	enc, _ := NewStream(5, 3)
	split := emptyBuffers(5)
	err := enc.Split(bytes.NewBuffer(data), toWriters(split), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	splits := toBytes(split)
	expect := len(data) / 5
	// Beware, if changing data size
	if split[0].Len() != expect {
		t.Errorf("unexpected size. expected %d, got %d", expect, split[0].Len())
	}

	err = enc.Split(bytes.NewBuffer([]byte{}), toWriters(emptyBuffers(3)), 0)
	if err != ErrShortData {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}

	buf := new(bytes.Buffer)
	err = enc.Join(buf, toReaders(toBuffers(splits)), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	joined := buf.Bytes()
	if !bytes.Equal(joined, data) {
		t.Fatal("recovered data does match original", joined[:8], data[:8], "... lengths:", len(joined), len(data))
	}

	err = enc.Join(buf, toReaders(emptyBuffers(2)), 0)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	bufs := toReaders(emptyBuffers(5))
	bufs[2] = nil
	err = enc.Join(buf, bufs, 0)
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}

	err = enc.Join(buf, toReaders(toBuffers(splits)), int64(len(data)+1))
	if err != ErrShortData {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}
}

func TestNewStream(t *testing.T) {
	tests := []struct {
		data, parity int
		err          error
	}{
		{10, 500, nil},
		{256, 256, nil},

		{0, 1, ErrInvShardNum},
		{1, 0, ErrInvShardNum},
		{257, 1, ErrInvShardNum},

		// overflow causes r.Shards to be negative
		{256, int(^uint(0) >> 1), errInvalidRowSize},
	}
	for _, test := range tests {
		_, err := NewStream(test.data, test.parity)
		if err != test.err {
			t.Errorf("New(%v, %v): expected %v, got %v", test.data, test.parity, test.err, err)
		}
	}
}