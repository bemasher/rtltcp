// This package provides a wrapper for the TCP protocol implemented by the rtl_tcp tool used with Realtek DVB-T based SDR's.
package rtltcp

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

var DongleMagic = [...]byte{'R', 'T', 'L', '0'}

// Contains dongle information and an embedded tcp connection to the spectrum server
type SDR struct {
	net.Conn
	Info DongleInfo
}

// Give an address of the form "0.0.0.0:1234" connects to the spectrum server
// at the given address or returns an error. The user is responsible for
// closing this connection.
func NewSDR(addr string) (sdr SDR, err error) {
	sdr.Conn, err = net.Dial("tcp", addr)
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

	err = binary.Read(sdr.Conn, binary.BigEndian, &sdr.Info)
	if err != nil {
		err = fmt.Errorf("Error getting dongle information: %s", err)
		return
	}

	if !sdr.Info.Valid() {
		err = fmt.Errorf("Invalid magic number: expected %q received %q", DongleMagic, sdr.Info.Magic)

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
	return d.Magic == DongleMagic
}

// Provides mapping of tuner value to tuner string.
type Tuner uint32

func (t Tuner) String() string {
	if t <= 5 {
		return []string{"UNKNOWN", "E4000", "FC0012", "FC0013", "FC2580", "R820T"}[t]
	}
	return "UNKNOWN"
}

func (sdr SDR) execute(cmd command) (err error) {
	return binary.Write(sdr.Conn, binary.BigEndian, cmd)
}

type command struct {
	command   uint8
	Parameter uint32
}

// Command constants defined in rtl_tcp.c
const (
	CenterFreq = iota + 1
	SampleRate
	TunerGainMode
	TunerGain
	FreqCorrection
	TunerIfGain
	TestMode
	AGCMode
	DirectSampling
	OffsetTuning
	RTLXtalFreq
	TunerXtalFreq
	GainByIndex
)

func (sdr SDR) SetCenterFreq(freq uint32) (err error) {
	return sdr.execute(command{CenterFreq, freq})
}

func (sdr SDR) SetSampleRate(sampleRate uint32) (err error) {
	return sdr.execute(command{SampleRate, sampleRate})
}

func (sdr SDR) SetGainMode(manual uint32) (err error) {
	return sdr.execute(command{TunerGainMode, manual})
}

func (sdr SDR) SetGain(gain uint32) (err error) {
	return sdr.execute(command{TunerGain, gain})
}

func (sdr SDR) SetGainByIndex(idx uint32) (err error) {
	if gain > sdr.Info.GainCount {
		return fmt.Errorf("invalid gain index: %d", gain)
	}
	return sdr.execute(command{GainByIndex, idx})
}

func (sdr SDR) SetFreqCorrection(ppm uint32) (err error) {
	return sdr.execute(command{FreqCorrection, ppm})
}

func (sdr SDR) SetTunerIfGain(stage, gain uint16) (err error) {
	return sdr.execute(command{TunerIfGain, (uint32(stage) << 16) | uint32(gain)})
}

func (sdr SDR) SetAGCMode(state bool) (err error) {
	if state {
		return sdr.execute(command{AGCMode, 1})
	}
	return sdr.execute(command{AGCMode, 0})
}

func (sdr SDR) SetDirectSampling(state bool) (err error) {
	if state {
		return sdr.execute(command{DirectSampling, 1})
	}
	return sdr.execute(command{DirectSampling, 0})
}

func (sdr SDR) SetOffsetTuning(state bool) (err error) {
	if state {
		return sdr.execute(command{OffsetTuning, 1})
	}
	return sdr.execute(command{OffsetTuning, 0})
}

func (sdr SDR) SetRTLXtalFreq(freq uint32) (err error) {
	return sdr.execute(command{RTLXtalFreq, freq})
}

func (sdr SDR) SetTunerXtalFreq(freq uint32) (err error) {
	return sdr.execute(command{TunerXtalFreq, freq})
}

func init() {
	log.SetFlags(log.Lshortfile)
}
