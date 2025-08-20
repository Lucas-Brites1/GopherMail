package utils

import "strings"

type Buffer_Config_t struct {
	Cap          int64
	Len          int64
	ResizeFactor int32
}

type Buffer_t struct {
	Buffer []string
	Config Buffer_Config_t
}

func NewBuffer(capacity int64, resizeFactor int32) *Buffer_t {
	return &Buffer_t{
		Buffer: make([]string, 0, capacity),
		Config: Buffer_Config_t{
			Cap:          capacity,
			Len:          0,
			ResizeFactor: resizeFactor,
		},
	}
}

func (b *Buffer_t) Write(strings ...string) {
	for _, str := range strings {
		b.Buffer = append(b.Buffer, str)
		b.Config.Len++

		if b.Config.Len >= b.Config.Cap {
			newCap := int64(float64(b.Config.Cap) * float64(b.Config.ResizeFactor))
			auxSlice := make([]string, 0, newCap)
			auxSlice = append(auxSlice, b.Buffer...)
			b.Buffer = nil
			b.Buffer = auxSlice
			b.Config.Cap = newCap
		}
	}
}

func (b *Buffer_t) Get() string {
	return strings.Join(b.Buffer, "")
}

func (b *Buffer_t) Reset() {
	b.Buffer = b.Buffer[:0]
	b.Config.Len = 0
}
