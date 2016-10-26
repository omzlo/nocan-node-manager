package nocan

func Bitmap64Fill(bitmap []byte, val byte) {
	for i := 0; i < 8; i++ {
		bitmap[i] = val
	}
}

func Bitmap64Set(bitmap []byte, index uint) {
	bitmap[7-(index>>3)] |= byte(1 << (index & 0x7))
}

func Bitmap64Clear(bitmap []byte, index uint) {
	bitmap[7-(index>>3)] &= ^byte(1 << (index & 0x7))
}

func Bitmap64Add(bitmap []byte, subset []byte) {
	for i := 0; i < 8; i++ {
		bitmap[i] |= subset[i]
	}
}

func Bitmap64Sub(bitmap []byte, subset []byte) {
	for i := 0; i < 8; i++ {
		bitmap[i] &= ^subset[i]
	}
}

func Bitmap64ToSlice(bitmap []byte) []byte {
	var i, j uint

	if len(bitmap) != 8 {
		return nil
	}
	slice := make([]byte, 0, 8)
	for i = 0; i < 8; i++ {
		for j = 0; j < 8; j++ {
			if (bitmap[7-i] & (1 << j)) != 0 {
				slice = append(slice, byte(i*8+j))
			}
		}
	}
	return slice
}
