package sps30_test

import (
	"bytes"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/MasandeM/sps30"
	"go.bug.st/serial"
)

func TestShdlcCRC(t *testing.T) {
	tests := []struct {
		addr    uint8
		cmd     uint8
		dataLen uint8
		data    []byte
		want    uint8
	}{
		{addr: 0, cmd: 1, dataLen: 0, data: []byte{}, want: 254},                       // Stop Measurement
		{addr: 0, cmd: 0, dataLen: 2, data: []byte{1, 3}, want: 0xF9},                  // Start Measurement
		{addr: 0, cmd: 0x56, dataLen: 0, data: []byte{}, want: 0xA9},                   // Start Fan Cleaning
		{addr: 0xFF, cmd: 0xFF, dataLen: 1, data: []byte{0xFF, 0xFF, 0xFF}, want: 0x1}, //non existing cmd
		{addr: 200, cmd: 100, dataLen: 4, data: []byte{50, 50, 50, 50}, want: 0x07},
		{addr: 200, cmd: 100, dataLen: 4, data: []byte{50, 50, 50, 50}, want: 0x07},
	}

	for _, test := range tests {
		if got := sps30.ShdlcCRC(test.addr+test.cmd, test.dataLen, test.data); got != test.want {
			t.Errorf("shdlcCRC(0x%x + 0x%x, 0x%x, 0x%x) = 0x%x", test.addr, test.cmd, test.want, test.data, got)
		}
	}
}

func TestStuffData(t *testing.T) {

	tests := []struct {
		dataLen        int
		inputData      []byte
		outputData     *[sps30.ShdlcFrameMaxTxFrameSize]byte
		inputPos       int
		outputPosWant  int
		outputDataWant []byte
	}{
		{dataLen: 1, inputData: []byte{0}, outputData: &[sps30.ShdlcFrameMaxTxFrameSize]byte{}, inputPos: 0, outputPosWant: 1, outputDataWant: []byte{0}},
		{dataLen: 1, inputData: []byte{0x7E}, outputData: &[sps30.ShdlcFrameMaxTxFrameSize]byte{}, inputPos: 0, outputPosWant: 2, outputDataWant: []byte{0x7D, 0x5E}},
		{dataLen: 4, inputData: []byte{0x7E, 0x7D, 0x11, 0x13}, outputData: &[sps30.ShdlcFrameMaxTxFrameSize]byte{}, inputPos: 0, outputPosWant: 8, outputDataWant: []byte{0x7D, 0x5E, 0x7D, 0x5D, 0x7D, 0x31, 0x7D, 0x33}},
		{dataLen: 4, inputData: []byte{0x34, 0x03, 0x00, 0xF1}, outputData: &[sps30.ShdlcFrameMaxTxFrameSize]byte{}, inputPos: 0, outputPosWant: 4, outputDataWant: []byte{0x34, 0x03, 0x00, 0xF1}},
	}
	for _, test := range tests {
		outPosGot := sps30.StuffData(test.dataLen, test.inputData, test.outputData, test.inputPos)
		if outPosGot != test.outputPosWant {
			t.Errorf("stuffData(0x%x, 0x%x, *[sps30.ShdlcFrameMaxTxFrameSize]byte, 0x%x) = 0x%x. Expected return value 0x%x", test.dataLen, test.inputData, test.inputPos, outPosGot, test.outputPosWant)
		}
		if !bytes.Equal((*test.outputData)[:test.outputPosWant], test.outputDataWant) {
			t.Errorf("stuffData(0x%x, 0x%x, *[sps30.ShdlcFrameMaxTxFrameSize]byte, 0x%x): output data '0x%x' does not match expected '0x%x'", test.dataLen, test.inputData, test.inputPos, (*test.outputData)[:test.outputPosWant], test.outputDataWant)
		}
	}
}

func Example() {

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	log.Println("Connecting to UART")

	uart, err := serial.Open("/dev/ttyUSB0", mode) //should be read from a config file or something
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully Connected")

	device := sps30.New(uart)

	//Create a struct that is passed to a function then is populated. th
	version_info := sps30.VersionInfo{}
	err = device.ReadVersion(&version_info)

	if err != nil {
		log.Fatal("Error reading version information: ", err)
	}

	fmt.Printf("FW: %d.%d, HW: %d, SHDLC: %d.%d\n",
		version_info.FirmwarMajor,
		version_info.FirmwarMinor,
		version_info.HardwarRevision,
		version_info.SHDLCMajor,
		version_info.SHDLCMinor)

	err = device.StartMeasurement()
	if err != nil {
		log.Fatal("error starting measurement")
	}

	measurement := sps30.Measurement{}
	for {

		err = device.ReadMeasurement(&measurement)
		if err != nil {
			fmt.Printf("[-] error reading measurement: %v", err)
		} else {
			fmt.Printf(`
measured values:
				%0.2f pm1.0
				%0.2f pm2.5
				%0.2f pm4.0
				%0.2f pm10.0
				%0.2f nc0.5
				%0.2f nc1.0
				%0.2f nc2.5
				%0.2f nc4.5
				%0.2f nc10.0
				%0.2f typical particle size
`,
				measurement.Mc1p0, measurement.Mc2p5, measurement.Mc4p0, measurement.Mc10p0, measurement.Nc0p5,
				measurement.Nc1p0, measurement.Nc2p5, measurement.Nc4p0, measurement.Nc10p0,
				measurement.TypicalParticleSize)
		}

		time.Sleep(time.Second)
	}
}
