// Package instance keeps Nexus to one process per user: later launches ask
// the running one, over a Unix socket, to show or hide its panel.
package instance

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
)

// maxSocketPath leaves room under the kernel's 108-byte sun_path limit.
const maxSocketPath = 100

// SocketPath is where the running Nexus listens. It falls back to the
// temporary directory when the runtime directory's path is too long for a
// socket.
func SocketPath() string {
	return socketPath(os.Getenv("XDG_RUNTIME_DIR"), os.TempDir(), os.Getuid())
}

func socketPath(runtimeDir, tempDir string, uid int) string {
	name := "plunz-nexus-" + strconv.Itoa(uid) + ".sock"
	if runtimeDir != "" {
		if p := filepath.Join(runtimeDir, name); len(p) <= maxSocketPath {
			return p
		}
	}
	return filepath.Join(tempDir, name)
}

// Toggle asks the running Nexus to show or hide its panel. It fails when
// no Nexus is running.
func Toggle(path string) error {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte{1})
	return err
}

// Listen calls toggle for every Toggle request until the returned close
// function runs. A stale socket from a crashed Nexus is replaced.
func Listen(path string, toggle func()) (func(), error) {
	_ = os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			var b [1]byte
			_, _ = conn.Read(b[:])
			_ = conn.Close()
			toggle()
		}
	}()
	return func() { _ = l.Close(); _ = os.Remove(path) }, nil
}
