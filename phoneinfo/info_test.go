package dev

import (
	"testing"
)

func TestPLATFORM(t *testing.T) {
	platform := Platform()
	if platform != PLATFORM_QUALCOMM && platform != PLATFORM_MTK && platform != PLATFORM_SPRD {
		t.Error("平台检测错误")
	}
}

func TestArch(t *testing.T) {
	arch := Arch()
	if arch != ARCH_ARM && arch != ARCH_ARM64 {
		t.Error("cpu架构检测错误")
	}
}
