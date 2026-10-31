package rtltcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"testing"
	"time"
)

func ExampleSDR() {
	var sdr SDR

	// Resolve address, this may be ip:port or hostname:port.
	addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("Error resolving address:", err)
	}

	// Connect to address and defer close.
	if err := sdr.Connect(addr); err != nil {
		log.Fatal("Error connecting:", err)
	}
	defer sdr.Close()

	// Print dongle info.
	fmt.Printf("%+v\n", sdr.Info)
	// Example: {Magic:"RTL0" Tuner:R820T GainCount:29}

	// Create an array of bytes for samples.
	buf := make([]byte, 16384)

	// Read the entire array. This is usually done in a loop.
	_, err = io.ReadFull(sdr, buf)
	if err != nil {
		log.Fatal("Error reading samples:", err)
	}

	// Do something with data in buf...
}

func TestMockServer(t *testing.T) {
	m := NewMockServer(nil)
	m.Info.Tuner = 2
	m.Info.GainCount = 4
	m.Samples = []byte{0x00, 0x01, 0xfe, 0xff}

	if err := m.Start(); err != nil {
		t.Fatal("starting mock server:", err)
	}
	defer m.Close()

	var sdr SDR
	if err := sdr.Connect(m.Addr()); err != nil {
		t.Fatal("connecting:", err)
	}

	t.Run("Handshake", func(t *testing.T) {
		if !sdr.Info.Valid() || sdr.Info.Tuner != m.Info.Tuner || sdr.Info.GainCount != m.Info.GainCount {
			t.Fatalf("dongle info: got %s want %s", sdr.Info, m.Info)
		}
	})

	t.Run("Samples", func(t *testing.T) {
		buf := make([]byte, sampleChunkSize)
		if _, err := io.ReadFull(sdr, buf); err != nil {
			t.Fatal("reading samples:", err)
		}
		for i := range buf {
			if want := m.Samples[i%len(m.Samples)]; buf[i] != want {
				t.Fatalf("sample %d: got %#x want %#x", i, buf[i], want)
			}
		}
	})

	t.Run("NoCallbacksAfterClose", func(t *testing.T) {
		hit := make(chan string, 1)
		m.OnCenterFreq(func(v uint32) { hit <- fmt.Sprint(v) })

		m.Close()

		select {
		case v := <-hit:
			t.Errorf("unexpected callback after close: %q", v)
		case <-time.After(100 * time.Millisecond):
		}
	})
}

func TestCommands(t *testing.T) {
	m := NewMockServer(nil)
	m.Info.Tuner = 2
	m.Info.GainCount = 4
	m.Samples = []byte{0x00, 0x01, 0xfe, 0xff}

	if err := m.Start(); err != nil {
		t.Fatal("starting mock server:", err)
	}
	defer m.Close()

	var sdr SDR
	if err := sdr.Connect(m.Addr()); err != nil {
		t.Fatal("connecting:", err)
	}

	for _, tc := range []struct {
		name  string
		setup func(done func())
		run   func() error
	}{
		{
			name: "CenterFreq",
			setup: func(done func()) {
				m.OnCenterFreq(func(u uint32) {
					if u != 123_456_789 {
						t.Errorf("want %d got %d", 123_456_789, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetCenterFreq(123_456_789) },
		}, {
			name: "SampleRate",
			setup: func(done func()) {
				m.OnSampleRate(func(u uint32) {
					if u != 1_234_567 {
						t.Errorf("want %d got %d", 1_234_567, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetSampleRate(1_234_567) },
		}, {
			name: "GainMode",
			setup: func(done func()) {
				m.OnGainMode(func(u uint32) {
					if u != 1 {
						t.Errorf("want %d got %d", 1, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetGainMode(true) },
		}, {
			name: "Gain",
			setup: func(done func()) {
				m.OnGain(func(u uint32) {
					if u != 30 {
						t.Errorf("want %d got %d", 30, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetGain(30) },
		}, {
			name: "FreqCorrection",
			setup: func(done func()) {
				m.OnFreqCorrection(func(u int32) {
					if u != -30 {
						t.Errorf("want %d got %d", -30, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetFreqCorrection(-30) },
		}, {
			name: "TunerIfGain",
			setup: func(done func()) {
				m.OnTunerIfGain(func(stage, gain int16) {
					if stage != 3 || gain != -30 {
						t.Errorf("want stage=%d gain=%d got stage=%d gain=%d", 3, -30, stage, gain)
					}
					done()
				})
			},
			run: func() error { return sdr.SetTunerIfGain(3, -30) },
		}, {
			name: "FreqCorrection",
			setup: func(done func()) {
				m.OnFreqCorrection(func(u int32) {
					if u != -30 {
						t.Errorf("want %d got %d", -30, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetFreqCorrection(-30) },
		}, {
			name: "TestMode",
			setup: func(done func()) {
				m.OnTestMode(func(u uint32) {
					if u != 1 {
						t.Errorf("want %d got %d", 1, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetTestMode(true) },
		}, {
			name: "AGCMode",
			setup: func(done func()) {
				m.OnAGCMode(func(u uint32) {
					if u != 1 {
						t.Errorf("want %d got %d", 1, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetAGCMode(true) },
		}, {
			name: "DirectSampling",
			setup: func(done func()) {
				m.OnDirectSampling(func(u uint32) {
					if u != 1 {
						t.Errorf("want %d got %d", 1, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetDirectSampling(true) },
		}, {
			name: "OffsetTuning",
			setup: func(done func()) {
				m.OnOffsetTuning(func(u uint32) {
					if u != 1 {
						t.Errorf("want %d got %d", 1, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetOffsetTuning(true) },
		}, {
			name: "RTLXtalFreq",
			setup: func(done func()) {
				m.OnRTLXtalFreq(func(u uint32) {
					if u != 32768 {
						t.Errorf("want %d got %d", 32768, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetRTLXtalFreq(32768) },
		}, {
			name: "TunerXtalFreq",
			setup: func(done func()) {
				m.OnTunerXtalFreq(func(u uint32) {
					if u != 32768 {
						t.Errorf("want %d got %d", 32768, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetTunerXtalFreq(32768) },
		}, {
			name: "GainByIndex",
			setup: func(done func()) {
				m.OnGainByIndex(func(u uint32) {
					if u != 3 {
						t.Errorf("want %d got %d", 3, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetGainByIndex(3) },
		}, {
			name: "BiasTee",
			setup: func(done func()) {
				m.OnBiasTee(func(u uint32) {
					if u != 1 {
						t.Errorf("want %d got %d", 1, u)
					}
					done()
				})
			},
			run: func() error { return sdr.SetBiasTee(true) },
		}, {
			name: "Unknown",
			setup: func(done func()) {
				m.OnUnknown(func(cmd byte, param uint32) {
					if cmd != 0x1F || param != 0x01020304 {
						t.Errorf("want cmd=0x%02X param=0x%08X got cmd=0x%02X param=0x%08X",
							0x1F, 0x01020304, cmd, param,
						)
					}
					done()
				})
			},
			run: func() error { return sdr.execute(command{0x1F, 0x01020304}) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, canc := context.WithTimeout(context.Background(), time.Second)
			defer canc()

			tc.setup(canc)
			if err := tc.run(); err != nil {
				t.Error(err)
			}

			<-ctx.Done()
			err := ctx.Err()
			switch {
			case err == nil:
			case errors.Is(err, context.Canceled):
			case errors.Is(err, context.DeadlineExceeded):
				t.Error(err)
			default:
				t.Error(err)
			}
		})
	}
}
