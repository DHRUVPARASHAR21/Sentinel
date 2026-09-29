// Package logging emits one structured JSON event per line without secrets.
package logging

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	w  io.Writer
	mu sync.Mutex
}

func New() *Logger { return &Logger{w: os.Stderr} }
func (l *Logger) Event(name string, fields map[string]any) {
	record := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "event": name}
	for k, v := range fields {
		record[k] = v
	}
	data, _ := json.Marshal(record)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.w.Write(append(data, '\n'))
}
