package main

var TunAddr = "10.100.0.10"

func main() {
	t := NewTun(TunAddr)
	t.Setup()
	t.Read()
}
