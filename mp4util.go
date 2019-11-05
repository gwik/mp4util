package mp4util

import (
	"bytes"
	"io"
	"io/ioutil"
)

type atom [4]byte

var (
	moov = atom{'m', 'o', 'o', 'v'}
	mvhd = atom{'m', 'v', 'h', 'd'}
)

// Returns the duration, in seconds, of the mp4 file at the provided filepath.
// If an error occurs, the error returned is non-nil.
func Duration(r io.Reader) (int, error) {

	// validate that we have a moov atom ?
	if _, err := findNextAtom(r, moov); err != nil {
		return 0, err
	}

	// start searching for the mvhd atom inside the moov atom.
	// The first child atom of the moov atom starts 8 bytes after the start of the moov atom.
	mvhdAtomLength, err := findNextAtom(r, mvhd)
	if err != nil {
		return 0, err
	}

	duration, err := durationFromMvhdAtom(r, mvhdAtomLength)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func skipN(r io.Reader, n int64) error {
	_, err := io.CopyN(ioutil.Discard, r, n)
	return err
}

// Finds the starting position of the atom of the given name.
func findNextAtom(r io.Reader, atomName atom) (int64, error) {
	buffer := make([]byte, 8)
	for {
		_, err := io.ReadFull(r, buffer)
		if err != nil {
			return 0, err
		}

		// The structure of an mp4 atom is:
		// 4 bytes - length of atom
		// 4 bytes - name of atom in ascii encoding
		// rest    - atom data
		lengthOfAtom := int64(convertBytesToInt(buffer[0:4]))
		if bytes.Equal(atomName[:], buffer[4:]) {
			return lengthOfAtom, nil
		}
		if err := skipN(r, lengthOfAtom); err != nil {
			return 0, err
		}
	}
	return 0, io.ErrUnexpectedEOF
}

// Returns the duration in seconds as given by the data in the mvhd atom starting at mvhdStart
// Returns non-nill error is there is an error.
func durationFromMvhdAtom(r io.Reader, mvhdLength int64) (int, error) {
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
	timescale := convertBytesToInt(buffer[0:4]) // This is in number of units per second
	durationInTimeScale := convertBytesToInt(buffer[4:])
	return int(durationInTimeScale) / int(timescale), nil
}

func convertBytesToInt(buf []byte) int {
	res := 0
	for i := len(buf) - 1; i >= 0; i-- {
		b := int(buf[i])
		shift := uint((len(buf) - 1 - i) * 8)
		res += b << shift
	}
	return res
}
