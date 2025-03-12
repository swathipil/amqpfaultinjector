package utils

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"iter"
	"os"
)

type BinLine struct {
	Label  string
	Packet []byte
}

func ParseBinFile(binFile string) iter.Seq2[BinLine, error] {
	return func(yield func(binLine BinLine, err error) bool) {
		reader, err := os.Open(binFile)

		if err != nil {
			return
		}

		bufReader := bufio.NewReader(reader)

		for {
			label, err := bufReader.ReadString(':')

			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				yield(BinLine{}, err)
				break
			}

			label = label[:len(label)-1]

			base64Data, err := bufReader.ReadString('\n')

			if err != nil {
				yield(BinLine{}, err)
				break
			}

			packetBytes, err := base64.StdEncoding.DecodeString(base64Data)

			if err != nil {
				yield(BinLine{}, err)
				break
			}

			if !yield(BinLine{Label: label, Packet: packetBytes}, nil) {
				break
			}
		}
	}
}
