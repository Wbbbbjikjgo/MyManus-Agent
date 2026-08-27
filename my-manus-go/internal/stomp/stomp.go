package stomp

import (
	"bytes"
	"fmt"
	"strings"
)

// Frame 表示一个 STOMP 帧
type Frame struct {
	Command string
	Headers map[string]string
	Body    []byte
}

// Parse 解析一个完整 STOMP 帧（data 以 \0 结尾）
func Parse(data []byte) (*Frame, error) {
	data = bytes.TrimSuffix(data, []byte{0})

	nl := bytes.IndexByte(data, '\n')
	if nl < 0 {
		return nil, fmt.Errorf("无效帧：缺少命令")
	}
	f := &Frame{Command: strings.TrimSpace(string(data[:nl])), Headers: map[string]string{}}

	rest := data[nl+1:]
	for {
		lineEnd := bytes.IndexByte(rest, '\n')
		if lineEnd < 0 {
			return nil, fmt.Errorf("无效帧：缺少头结束符")
		}
		line := rest[:lineEnd]
		rest = rest[lineEnd+1:]
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if len(line) == 0 {
			break // 空行，rest 即 body
		}
		if parts := bytes.SplitN(line, []byte{':'}, 2); len(parts) == 2 {
			f.Headers[string(parts[0])] = string(parts[1])
		}
	}
	f.Body = rest
	return f, nil
}

// Encode 将帧序列化为 STOMP 文本（含末尾 \0）
func (f *Frame) Encode() []byte {
	var b bytes.Buffer
	b.WriteString(f.Command)
	b.WriteByte('\n')
	for k, v := range f.Headers {
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.Write(f.Body)
	b.WriteByte(0)
	return b.Bytes()
}
