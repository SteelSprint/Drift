package statestore

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	"drift/internal/fileio"
)

// baselinesFileName is the name of the gob-encoded packfile inside .drift/.
const baselinesFileName = "baselines.bin"

// D! id=pbase range-start

// ErrBaselineHashMismatch is retained for backward-compatibility with the
// pre-packed format; the new BaselineStore does not verify sha1(content)==hash
// because spec canonical hashes legitimately mismatch raw content (refs stripped).
var ErrBaselineHashMismatch = errors.New("baseline content hash does not match declared hash")

// BaselineStore manages content-addressed baseline content packed in a single
// .drift/baselines.bin file (gob-encoded map[string]string). Stateless between
// calls — each Read/Write/Delete loads and persists via the Session.
type BaselineStore struct{}

// NewBaselineStore returns a fresh BaselineStore. No path argument: the
// packfile location is owned by the Session (always .drift/baselines.bin).
func NewBaselineStore() *BaselineStore {
	return &BaselineStore{}
}

// Write stores content for hash in the packfile via the Session. If hash
// already has identical content, Write is a no-op (dedup). The packfile is
// loaded, updated, marshaled, and atomically rewritten.
func (b *BaselineStore) Write(sess *fileio.Session, hash, content string) error {
	data, err := b.loadAll(sess)
	if err != nil {
		return err
	}
	if existing, ok := data[hash]; ok && existing == content {
		return nil
	}
	data[hash] = content
	return b.saveAll(sess, data)
}

// Read returns the baseline content for hash. Returns ("", false) when the
// packfile is missing or hash is not present.
func (b *BaselineStore) Read(sess *fileio.Session, hash string) (string, bool) {
	data, err := b.loadAll(sess)
	if err != nil {
		return "", false
	}
	content, ok := data[hash]
	return content, ok
}

// Delete removes the entry for hash from the packfile. Missing hash is a
// no-op (returns nil).
func (b *BaselineStore) Delete(sess *fileio.Session, hash string) error {
	data, err := b.loadAll(sess)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, ok := data[hash]; !ok {
		return nil
	}
	delete(data, hash)
	return b.saveAll(sess, data)
}

// loadAll reads and unmarshals the packfile. Returns an empty map if the
// file does not exist (fresh project).
func (b *BaselineStore) loadAll(sess *fileio.Session) (map[string]string, error) {
	raw, err := sess.Read(baselinesFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var data map[string]string
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode %s: %w", baselinesFileName, err)
	}
	if data == nil {
		data = map[string]string{}
	}
	return data, nil
}

// saveAll marshals the map and atomic-writes it via the Session.
func (b *BaselineStore) saveAll(sess *fileio.Session, data map[string]string) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("encode %s: %w", baselinesFileName, err)
	}
	return sess.Write(baselinesFileName, buf.Bytes())
}

// D! id=pbase range-end
