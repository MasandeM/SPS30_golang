package sps30_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/MasandeM/sps30"
	"go.bug.st/serial"
)

type fakeUart struct {
	Data *bytes.Buffer
}

func (f fakeUart) UartBuffer() []byte {

	return (*f.Data).Bytes()
}

func (f fakeUart) Write(p []byte) (n int, err error) {
	(*f.Data).Write(p)
	return 0, nil
}

func (f fakeUart) Break(time.Duration) error {
	return nil
}

func (f fakeUart) SetMode(mode *serial.Mode) error {
	return nil
}
func (f fakeUart) Read(p []byte) (n int, err error) {
	(*f.Data).Read(p)
	return len(p), nil
}

func (f fakeUart) Drain() error {
	return nil
}

func (f fakeUart) ResetInputBuffer() error {
	return nil
}

func (f fakeUart) ResetOutputBuffer() error {
	return nil
}

func (f fakeUart) SetDTR(dtr bool) error {
	return nil
}

func (f fakeUart) SetRTS(rts bool) error {
	return nil
}

func (f fakeUart) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &serial.ModemStatusBits{}, nil
}

func (f fakeUart) SetReadTimeout(t time.Duration) error {
	return nil
}

func (uart fakeUart) Close() error {
	return nil
}

func TestWakeup(t *testing.T) {
	tests := []struct {
		uartBuffer []byte
	}{
		{uartBuffer: []byte{0x7e, 0x00, 0x7d, 0x31, 0x43, 0x00, 0xab, 0x7e}},
	}
	for _, test := range tests {
		mockUart := fakeUart{Data: bytes.NewBuffer(test.uartBuffer)}
		device := sps30.New(mockUart)
		err := device.Wakeup()

		if err != nil {
			t.Errorf("Wakeup(): %v", err)
		}
	}
}
func TestReadVersion(t *testing.T) {

	tests := []struct {
		versionInfo *sps30.VersionInfo
		uartBuffer  []byte
	}{
		{versionInfo: &sps30.VersionInfo{}, uartBuffer: []byte{0x7e, 0x00, 0xd1, 0x00, 0x07, 0x02, 0x03, 0x00, 0x07, 0x00, 0x02, 0x00, 0x19, 0x7e}},
	}

	for _, test := range tests {
		mockUart := fakeUart{Data: bytes.NewBuffer(test.uartBuffer)}
		device := sps30.New(mockUart)
		err := device.ReadVersion(test.versionInfo)

		if err != nil {
			t.Errorf("ReadVersion(*versionInfo): %v", err)
		}
	}

}

func TestStartMeasurement(t *testing.T) {
	tests := []struct {
		uartBuffer []byte
	}{
		{uartBuffer: []byte{0x7e, 0x00, 0x00, 0x43, 0x00, 0xbc, 0x7e}},
	}
	for _, test := range tests {
		mockUart := fakeUart{Data: bytes.NewBuffer(test.uartBuffer)}
		device := sps30.New(mockUart)
		err := device.StartMeasurement()

		if err != nil {
			t.Errorf("StartMeasurement(): %v", err)
		}
	}
}

func TestReadMeasurement(t *testing.T) {
	tests := []struct {
		measurement *sps30.Measurement
		uartBuffer  []byte
	}{
		{measurement: &sps30.Measurement{}, uartBuffer: []byte{0x7e, 0x00, 0x03, 0x00, 0x28, 0x3d, 0x20, 0x01, 0xe3, 0x3d, 0x5a, 0xde, 0xcf, 0x3d, 0x81, 0xcc, 0x53, 0x3d, 0x8c, 0x48, 0xae, 0x3e, 0x73, 0x39, 0x7d, 0x5e, 0x3e, 0x97, 0xb9, 0x03, 0x3e, 0x9e, 0x8b, 0x9f, 0x3e, 0x9f, 0xf1, 0x71, 0x3e, 0xa0, 0x4e, 0x18, 0x3f, 0x36, 0x50, 0x12, 0x5a, 0x7e}},
	}
	for _, test := range tests {
		mockUart := fakeUart{Data: bytes.NewBuffer(test.uartBuffer)}
		device := sps30.New(mockUart)
		err := device.ReadMeasurement(test.measurement)

		if err != nil {
			t.Errorf("StartMeasurement(): %v", err)
		}
	}
}

func TestShdlcTx(t *testing.T) {
	tests := []struct {
		addr    uint8
		cmd     uint8
		dataLen uint8
		data    []byte
		want    string
	}{
		{addr: 0x00, cmd: 0x00, dataLen: 0x02, data: []byte{0x01, 0x03}, want: "7e0000020103f97e"}, // MOSI Start measurement
		{addr: 0x00, cmd: 0x01, dataLen: 0x00, data: []byte{}, want: "7e000100fe7e"},               // MOSI Stop measurement
		{addr: 0x00, cmd: 0x03, dataLen: 0x00, data: []byte{}, want: "7e000300fc7e"},               // MOSI Read Measurement Value
		{addr: 0x00, cmd: 0x10, dataLen: 0x00, data: []byte{}, want: "7e001000ef7e"},               // MOSI Sleep
		{addr: 0x00, cmd: 0xd3, dataLen: 0x00, data: []byte{}, want: "7e00d3002c7e"},               // MOSI Device reset
	}

	for _, test := range tests {
		mockUart := fakeUart{Data: new(bytes.Buffer)}
		device := sps30.New(mockUart)
		device.ShdlcTx(test.addr, test.cmd, test.dataLen, test.data)

		got := hex.EncodeToString(mockUart.UartBuffer())
		if got != test.want {
			t.Errorf("ShdlcTx(0x%x, 0x%x, 0x%x, 0x%x) = 0x%v", test.addr, test.cmd, test.dataLen, test.data, got)
		}

	}
}

func TestShdlcRx(t *testing.T) {
	tests := []struct {
		maxDataLen int
		rxHeader   *sps30.ShdlcRxHeader
		data       *[]byte
		uartBuffer []byte
		want       []byte
	}{
		{maxDataLen: 0, rxHeader: &sps30.ShdlcRxHeader{}, data: &[]byte{}, uartBuffer: []byte{0x7e, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x7e}, want: []byte{}},                                                                                                            // MISO Read measurement
		{maxDataLen: 7, rxHeader: &sps30.ShdlcRxHeader{}, data: &[]byte{0, 0, 0, 0, 0, 0, 0}, uartBuffer: []byte{0x7e, 0x00, 0xd1, 0x00, 0x07, 0x02, 0x7D, 0x31, 0x00, 0x07, 0x00, 0x02, 0x00, 0x0b, 0x7e}, want: []byte{0x02, 0x11, 0x00, 0x07, 0x00, 0x02, 0x00}}, // MISO Read Version, with byte stuffing
		{maxDataLen: 5, rxHeader: &sps30.ShdlcRxHeader{}, data: &[]byte{0, 0, 0, 0, 0}, uartBuffer: []byte{0x7e, 0x00, 0xd2, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x28, 0x7e}, want: []byte{0x00, 0x00, 0x00, 0x00, 0x00}},                                     // MISO Read Device Status Register

	}

	for _, test := range tests {
		mockUart := fakeUart{Data: bytes.NewBuffer(test.uartBuffer)}
		device := sps30.New(mockUart)
		err := device.ShdlcRx(test.maxDataLen, test.rxHeader, test.data)

		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal((*test.data), test.want) {
			t.Errorf("shdlcRx(0x%x, 0x%x, 0x%x) = 0x%x", test.maxDataLen, test.rxHeader, test.data, (*test.data))
		}
	}
}
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
		{addr: 0xFF, cmd: 0xFF, dataLen: 1, data: []byte{0xFF, 0xFF, 0xFF}, want: 0x1}, // non existing cmd
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
		outputPos := sps30.StuffData(test.dataLen, test.inputData, test.outputData, test.inputPos)
		if outputPos != test.outputPosWant {
			t.Errorf("stuffData(0x%x, 0x%x, *[sps30.ShdlcFrameMaxTxFrameSize]byte, 0x%x) = 0x%x. Expected return value 0x%x", test.dataLen, test.inputData, test.inputPos, outputPos, test.outputPosWant)
		}
		if !bytes.Equal((*test.outputData)[:test.outputPosWant], test.outputDataWant) {
			t.Errorf("stuffData(0x%x, 0x%x, *[sps30.ShdlcFrameMaxTxFrameSize]byte, 0x%x): output data '0x%x' does not match expected '0x%x'", test.dataLen, test.inputData, test.inputPos, (*test.outputData)[:test.outputPosWant], test.outputDataWant)
		}
	}
}

func TestUnstuffByte(t *testing.T) {

	createUint8Pointer := func(val uint8) *uint8 {
		return &val
	}

	tests := []struct {
		data           []byte
		index          int
		outputData     *uint8
		outputPosWant  int
		outputDataWant uint8
	}{
		{data: []byte{0x7d, 0x31, 0x03, 0x05}, index: 0, outputData: createUint8Pointer(0), outputPosWant: 2, outputDataWant: 0x11},
		{data: []byte{0x7d, 0x31, 0x03, 0x05}, index: 2, outputData: createUint8Pointer(0), outputPosWant: 3, outputDataWant: 0x03},
		{data: []byte{0x7d, 0x31, 0x7D, 0x33}, index: 2, outputData: createUint8Pointer(0), outputPosWant: 4, outputDataWant: 0x13},
		{data: []byte{0xFF, 0x31, 0x03, 0x05}, index: 0, outputData: createUint8Pointer(0), outputPosWant: 1, outputDataWant: 0xFF},
	}
	for _, test := range tests {
		outputPos := sps30.UnstuffByte(test.data, test.index, test.outputData)
		if outputPos != test.outputPosWant {
			t.Errorf("unstuffByte(0x%x, 0x%x, []) = 0x%x. Expected return value 0x%x", test.data, test.index, outputPos, test.outputPosWant)
		}
		if *test.outputData != test.outputDataWant {
			t.Errorf("unstuffByte(0x%x, 0x%x, []): output data '0x%x' does not match expected '0x%x'", test.data, test.index, *test.outputData, test.outputDataWant)
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
