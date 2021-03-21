package dev

import (
	"fmt"
	"log"
	"os/exec"
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

func SetProp(name, value string) error {
	cmd := exec.Command("setprop " + name + " " + value)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func GetProp(name string) string {
	cmd := exec.Command("getprop " + name)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return cmd.Stdout
}
