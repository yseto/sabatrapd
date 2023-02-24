package charset

import (
	"bufio"
	"bytes"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func transformShiftJIS(b []byte) (string, error) {
	scanner := bufio.NewScanner(transform.NewReader(bytes.NewBuffer(b), japanese.ShiftJIS.NewDecoder()))
	var str string
	for scanner.Scan() {
		str += scanner.Text()
	}
	return str, scanner.Err()
}
