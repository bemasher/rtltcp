// This package provides a wrapper for the TCP protocol implemented by the rtl_tcp tool used with Realtek DVB-T based SDR's.
package rtltcp

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

var dongleMagic = [...]byte{'R', 'T', 'L', '0'}

// Contains dongle information and an embedded tcp connection to the spectrum server
type SDR struct {
	*net.TCPConn
	Info DongleInfo
}

// Give an address of the form "0.0.0.0:1234" connects to the spectrum server
// at the given address or returns an error. The user is responsible for
// closing this connection.
func (sdr *SDR) Connect(addr *net.TCPAddr) (err error) {
	sdr.TCPConn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		err = fmt.Errorf("Error connecting to spectrum server: %s", err)
		return
	}

	// If we exit this function due to an error, close the connection.
	defer func() {
		if err != nil {
			sdr.Close()
		}
	}()

	err = binary.Read(sdr.TCPConn, binary.BigEndian, &sdr.Info)
	if err != nil {
		err = fmt.Errorf("Error getting dongle information: %s", err)
		return
	}

	if !sdr.Info.Valid() {
		err = fmt.Errorf("Invalid magic number: expected %q received %q", dongleMagic, sdr.Info.Magic)
	}

	return
}

// Contains the Magic number, tuner information and the number of valid gain values.
type DongleInfo struct {
	Magic     [4]byte
	Tuner     Tuner
	GainCount uint32 // Useful for setting gain by index
}

func (d DongleInfo) String() string {
	return fmt.Sprintf("{Magic:%q Tuner:%s GainCount:%d}", d.Magic, d.Tuner, d.GainCount)
}

// Checks that the magic number received matches the expected byte string 'RTL0'.
func (d DongleInfo) Valid() bool {
	return d.Magic == dongleMagic
}

// Provides mapping of tuner value to tuner string.
type Tuner uint32

func (t Tuner) String() string {
	switch t {
	case 1:
		return "E4000"
	case 2:
		return "FC0012"
	case 3:
		return "FC0013"
	case 4:
		return "FC2580"
	case 5:
		return "R820T"
	case 6:
		return "R828D"
	}
	return "UNKNOWN"
}

func (sdr SDR) execute(cmd command) (err error) {
	return binary.Write(sdr.TCPConn, binary.BigEndian, cmd)
}

type command struct {
	command   uint8
	Parameter uint32
}

// Command constants defined in rtl_tcp.c
const (
	centerFreq = iota + 1
	sampleRate
	tunerGainMode
	tunerGain
	freqCorrection
	tunerIfGain
	testMode
	agcMode
	directSampling
	offsetTuning
	rtlXtalFreq
	tunerXtalFreq
	gainByIndex
)

// Set the center frequency in Hz.
func (sdr SDR) SetCenterFreq(freq uint32) (err error) {
	return sdr.execute(command{centerFreq, freq})
}

// Set the sample rate in Hz.
func (sdr SDR) SetSampleRate(rate uint32) (err error) {
	return sdr.execute(command{sampleRate, rate})
}

// Set gain in tenths of dB. (197 => 19.7dB)
func (sdr SDR) SetGain(gain uint32) (err error) {
	return sdr.execute(command{tunerGain, gain})
}

// Set the Tuner AGC, true to enable.
func (sdr SDR) SetGainMode(state bool) (err error) {
	if state {
		return sdr.execute(command{tunerGainMode, 0})
	}
	return sdr.execute(command{tunerGainMode, 1})
}

// Set gain by index, must be <= DongleInfo.GainCount
func (sdr SDR) SetGainByIndex(idx uint32) (err error) {
	if idx > sdr.Info.GainCount {
		return fmt.Errorf("invalid gain index: %d", idx)
	}
	return sdr.execute(command{gainByIndex, idx})
}

// Set frequency correction in ppm.
func (sdr SDR) SetFreqCorrection(ppm uint32) (err error) {
	return sdr.execute(command{freqCorrection, ppm})
}

// Set tuner intermediate frequency stage and gain.
func (sdr SDR) SetTunerIfGain(stage, gain uint16) (err error) {
	return sdr.execute(command{tunerIfGain, (uint32(stage) << 16) | uint32(gain)})
}

// Set test mode, true for enabled.
func (sdr SDR) SetTestMode(state bool) (err error) {
	if state {
		return sdr.execute(command{testMode, 1})
	}
	return sdr.execute(command{testMode, 0})
}

// Set RTL AGC mode, true for enabled.
func (sdr SDR) SetAGCMode(state bool) (err error) {
	if state {
		return sdr.execute(command{agcMode, 1})
	}
	return sdr.execute(command{agcMode, 0})
}

// Set direct sampling mode.
func (sdr SDR) SetDirectSampling(state bool) (err error) {
	if state {
		return sdr.execute(command{directSampling, 1})
	}
	return sdr.execute(command{directSampling, 0})
}

// Set offset tuning, true for enabled.
func (sdr SDR) SetOffsetTuning(state bool) (err error) {
	if state {
		return sdr.execute(command{offsetTuning, 1})
	}
	return sdr.execute(command{offsetTuning, 0})
}

// Set RTL xtal frequency.
func (sdr SDR) SetRTLXtalFreq(freq uint32) (err error) {
	return sdr.execute(command{rtlXtalFreq, freq})
}

// Set tuner xtal frequency.
func (sdr SDR) SetTunerXtalFreq(freq uint32) (err error) {
	return sdr.execute(command{tunerXtalFreq, freq})
}

func init() {
	log.SetFlags(log.Lshortfile)
}
