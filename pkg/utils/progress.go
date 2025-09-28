package utils

import (
	"fmt"
	"strings"
)

func PrintProgress(section string, current int, total int, description string) {
	nbBlocks := 100

	// Handle edge cases to prevent negative values
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}

	blocks := int(float64(current) * float64(nbBlocks) / float64(total))
	if blocks < 0 {
		blocks = 0
	}
	if blocks > nbBlocks {
		blocks = nbBlocks
	}

	percentage := current * 100 / total
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	bar := fmt.Sprintf("\r[%s%s] %d%% (%d/%d) | %s",
		strings.Repeat("=", blocks),
		strings.Repeat(" ", nbBlocks-blocks),
		percentage,
		current,
		total,
		description)
	fmt.Print(bar)
}
