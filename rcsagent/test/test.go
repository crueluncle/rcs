package main

import (
	"fmt"
	"os/exec"
)

func main() {

	cmd := "cat /proc/cpuinfo | egrep '^model name' | uniq | awk '{print substr($0, index($0,$4))}'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Println("Failed to execute command: ", cmd)
	}
	fmt.Println(string(out))

}
