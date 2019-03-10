package main

import (
	"fmt"
	"log"
	"os/exec"
)

func Exec(cmd string, args ...interface{}) error {
	c := exec.Command("sh", "-c", fmt.Sprintf(cmd, args...))
	out, err := c.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}
