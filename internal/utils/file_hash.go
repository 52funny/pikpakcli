package utils

import (
	"crypto/sha1"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
)

var BufSize = 1 << 16

// number of the calculate routine
var CalculateRoutine = 1 << 3

func getChunkSize(size int64) int64 {
	if size > 0 && size < 0x8000000 {
		return 0x40000
	}
	if size >= 0x8000000 && size < 0x10000000 {
		return 0x80000
	}
	if size <= 0x10000000 || size > 0x20000000 {
		return 0x200000
	}
	return 0x100000
}

// func FileSha1(path string) string {
// 	file, err := os.Open(path)
// 	if err != nil {
// 		return ""
// 	}
// 	defer file.Close()

// 	resHash := sha1.New()

// 	state, _ := file.Stat()
// 	chunk := getChunkSize(state.Size())

// 	buf := make([]byte, BufSize)

// 	var total int64 = 0
// 	var partHash hash.Hash

// LABEL:
// 	for {
// 		partHash = sha1.New()
// 		total = 0
// 		for {
// 			n, err := file.Read(buf)
// 			if err != nil {
// 				if err == io.EOF {
// 					break LABEL
// 				}
// 				return ""
// 			}
// 			partHash.Write(buf[:n])
// 			total += int64(n)

// 			if total >= chunk {
// 				break
// 			}
// 		}
// 		resHash.Write(partHash.Sum(nil))
// 	}
// 	if total > 0 {
// 		resHash.Write(partHash.Sum(nil))
// 	}
// 	checksum := fmt.Sprintf("%x", resHash.Sum(nil))
// 	return checksum
// }

type segmentInfo struct {
	id     int64
	offset int64
}
type segmentData struct {
	id   int64
	data []byte
}

func calculate(file *os.File, chunkSize int64, ch chan segmentInfo, outCh chan segmentData) {
	buf := make([]byte, BufSize)
	for {
		info, ok := <-ch
		// fmt.Println(info, ok)
		if !ok {
			break
		}
		partHash := sha1.New()
		offset := info.offset
		total := int64(0)
		for {

			// ReadAt not similar with Read
			n, err := file.ReadAt(buf, offset)
			offset += int64(n)
			total += int64(n)

			// write buf to hash to calculate the sha1
			partHash.Write(buf[:n])

			if total >= chunkSize {
				break
			}

			if err != nil && err == io.EOF {
				break
			}

		}
		outCh <- segmentData{id: info.id, data: partHash.Sum(nil)}
	}
}

func FileSha1(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	resHash := sha1.New()

	state, _ := file.Stat()
	chunk := getChunkSize(state.Size())

	// segment length
	segmentLen := int64(math.Ceil(float64(state.Size()) / float64(chunk)))

	inCh := make(chan segmentInfo, segmentLen)
	outCh := make(chan segmentData, CalculateRoutine)

	// start go routine
	for i := 0; i < CalculateRoutine; i++ {
		go calculate(file, chunk, inCh, outCh)
	}

	// send info data to go routine
	for i := int64(0); i < segmentLen; i++ {
		inCh <- segmentInfo{id: i, offset: chunk * i}
	}
	close(inCh)

	// collect data
	result := make([]segmentData, 0)
	for i := int64(0); i < segmentLen; i++ {
		result = append(result, <-outCh)
	}
	// close the channel
	close(outCh)

	// sort by id asc
	sort.Slice(result, func(i, j int) bool {
		return result[i].id < result[j].id
	})

	// fmt.Printf("%d %x\n", result[1].id, result[1].data)
	// fmt.Printf("%d %x\n", result[2].id, result[2].data)
	// calculate file sha1
	for i := 0; i < len(result); i++ {
		resHash.Write(result[i].data)
	}
	checksum := fmt.Sprintf("%x", resHash.Sum(nil))
	return checksum
}
