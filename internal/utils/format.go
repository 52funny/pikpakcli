package utils

import "strconv"

var storageUnits = [...]string{"B", "KB", "MB", "GB", "TB", "PB"}

func FormatStorage(sizeText string, human bool) string {
	if !human {
		return sizeText
	}
	size, err := strconv.ParseFloat(sizeText, 64)
	if err != nil {
		return sizeText
	}

	unit := 0
	for size >= 1024 && unit < len(storageUnits)-1 {
		size /= 1024
		unit++
	}

	if size == float64(int64(size)) {
		return strconv.FormatFloat(size, 'f', 0, 64) + storageUnits[unit]
	}

	return strconv.FormatFloat(size, 'f', 2, 64) + storageUnits[unit]
}
