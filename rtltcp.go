package rtltcp

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

var DongleMagic = [...]byte{'R', 'T', 'L', '0'}

type SDR struct {
	net.Conn
	Info DongleInfo
}

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

type DongleInfo struct {
	Magic     [4]byte
	Tuner     Tuner
	GainCount uint32
}

func (d DongleInfo) String() string {
	return fmt.Sprintf("{Magic:%q Tuner:%s GainCount:%d}", d.Magic, d.Tuner, d.GainCount)
}

func (d DongleInfo) Valid() bool {
	return d.Magic == DongleMagic
}

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

func (sdr SDR) SetFreq(freq uint32) (err error) {
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

func (sdr SDR) SetGainByIndex(idx uint32) (err error) {
	return sdr.execute(command{GainByIndex, idx})
}

type IQ struct {
	I, Q byte
}

func (iq *IQ) Normalize() {
	if iq.I >= 128 {
		iq.I += 128
	} else {
		iq.I = 128 - iq.I
	}

	if iq.Q >= 128 {
		iq.Q += 128
	} else {
		iq.Q = 128 - iq.Q
	}
}

func (iq IQ) Complex() complex128 {
	return complex((float64(iq.I)-127.5)/127.5, (float64(iq.Q)-127.5)/127.5)
}

func (sdr SDR) Sample(samples []complex128) (err error) {
	buf := make([]IQ, len(samples))

	err = sdr.SampleIQ(buf)
	if err != nil {
		return err
	}

	for k, iq := range buf {
		samples[k] = iq.Complex()
	}

	return
}

func (sdr SDR) SampleIQ(samples []IQ) (err error) {
	err = binary.Read(sdr.Conn, binary.BigEndian, &samples)

	if err != nil {
		return fmt.Errorf("Error reading samples: %s", err)
	}

	return err
}

func init() {
	log.SetFlags(log.Lshortfile)
}
