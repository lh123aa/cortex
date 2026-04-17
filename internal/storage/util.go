package storage

import (
	"bytes"
	"encoding/binary"
)

// Float32ArrayToBytes 将浮点数组转为二进制（极大加速存储与读取性能）
func Float32ArrayToBytes(arr []float32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, arr)
	return buf.Bytes()
}

// BytesToFloat32Array 将底层二进制快速映射回浮点数组
func BytesToFloat32Array(b []byte) []float32 {
	arr := make([]float32, len(b)/4)
	buf := bytes.NewReader(b)
	_ = binary.Read(buf, binary.LittleEndian, &arr)
	return arr
}
