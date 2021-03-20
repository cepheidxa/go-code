package phoneinfo

import (
	"fmt"
)

const (
	PLATFORM_QUALCOMM = 0
	PLATFORM_MTK      = 1
	PLATFORM_SPRD     = 2

	ARCH_ARM   = 0
	ARCH_ARM64 = 1
)

func Platform() int {
	return PLATFORM_QUALCOMM
}

func Arch() int {
	return ARCH_ARM
}

func T() {
	fmt.Println("T")
}
