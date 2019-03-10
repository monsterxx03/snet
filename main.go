package main

var TunAddr = "10.100.0.1"

func main() {
	t := NewTun(TunAddr)
	t.Setup()
	t.Read()
}
