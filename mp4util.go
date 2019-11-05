package mp4util

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"time"
)

type atom [4]byte

var (
	moov = atom{'m', 'o', 'o', 'v'}
	mvhd = atom{'m', 'v', 'h', 'd'}
)

// Returns the duration, in seconds, of the mp4 file at the provided filepath.
// If an error occurs, the error returned is non-nil.
func Duration(r io.Reader) (time.Duration, error) {

	// find the moov atom which is a container for the mvhd atom.
	if _, err := findNextAtom(r, moov); err != nil {
		return 0, err
	}

	// start searching for the mvhd atom inside the moov atom.
	// The first child atom of the moov atom starts 8 bytes after the start of the moov atom.
	mvhdAtomLength, err := findNextAtom(r, mvhd)
	if err != nil {
		return 0, err
	}

	if mvhdAtomLength < 0 {
		return 0, io.ErrUnexpectedEOF
	}

	return durationFromMvhdAtom(r, mvhdAtomLength)
}

func skipN(r io.Reader, n int64) error {
	_, err := io.CopyN(ioutil.Discard, r, n)
	return err
}

// Finds the starting position of the atom of the given name and return the
// size of the atom.
// -1 size with no error means that atom data continues is until EOF.
func findNextAtom(r io.Reader, atomName atom) (int64, error) {
	buffer := make([]byte, 8)
	for {
		n, err := io.ReadFull(r, buffer)
		if err != nil {
			return 0, err
		}

		// The structure of an mp4 atom is:
		// 4 bytes - length of atom
		// 4 bytes - name of atom in ascii encoding
		// rest    - atom data
		atomSize := binary.BigEndian.Uint32(buffer[0:4])
		var realSize int64

		switch atomSize {
		case 0x0:
			if bytes.Equal(atomName[:], buffer[4:]) {
				// atom continues until EOF
				return -1, nil
			} else {
				// atom continues until EOF but it's not the one we are looking
				// for.
				return 0, io.ErrUnexpectedEOF
			}
		case 0x1: // extended (64 bit) size
			sizebuf := make([]byte, 8)
			_, err := io.ReadFull(r, sizebuf)
			if err != nil {
				return 0, err
			}
			n += 8
			realSize = int64(binary.BigEndian.Uint64(sizebuf))
		default:
			realSize = int64(atomSize)
		}

		if bytes.Equal(atomName[:], buffer[4:]) {
			return realSize, nil
		}
		if err := skipN(r, realSize-int64(n)); err != nil {
			return 0, err
		}
	}
	return 0, io.ErrUnexpectedEOF
}

// Returns the duration in seconds as given by the data in the mvhd atom starting at mvhdStart
// Returns non-nill error is there is an error.
func durationFromMvhdAtom(r io.Reader, mvhdLength int64) (time.Duration, error) {
	// The timescale field starts at the 21st byte of the mvhd atom
	if err := skipN(r, 20); err != nil {
		return 0, err
	}

	buffer := make([]byte, 8)
	if _, err := io.ReadFull(r, buffer); err != nil {
		return 0, err
	}

	// The timescale is bytes 21-24.
	// The duration is bytes 25-28
	timescale := binary.BigEndian.Uint32(buffer[0:4]) // This is in number of units per second
	if timescale == 0 {
		timescale = 600
	}
	durationInTimeScale := binary.BigEndian.Uint32(buffer[4:])
	return time.Duration(durationInTimeScale) * time.Second / time.Duration(timescale), nil
}
