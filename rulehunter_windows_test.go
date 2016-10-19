package main

import "testing"

func TestSubMain_interrupt(t *testing.T) {
	t.Skip("This test isn't implemented for Windows yet")
}

/*************************************
 *  Helper functions
 *************************************/

func startService(t *testing.T, serviceName string) {
	runCmd(t, "net", "start", serviceName)
}

func stopService(t *testing.T, serviceName string) {
	runCmd(t, "net", "stop", serviceName)
}