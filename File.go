package u

import (
	"bufio"
	"bytes"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func RunCommand(command string, args ...string) ([]string, error) {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	outs := make([]string, 0)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(io.MultiReader(stdout, stderr))
	for {
		lineBuf, _, err2 := reader.ReadLine()

		if err2 != nil || io.EOF == err2 {
			break
		}
		line := strings.TrimRight(string(lineBuf), "\r\n")
		outs = append(outs, line)
	}

	_ = cmd.Wait()
	return outs, nil
}

func ReadFileLines(fileName string) ([]string, error) {
	outs := make([]string, 0)
	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0400)
	if err != nil {
		return outs, err
	}

	inputReader := bufio.NewReader(fd)
	for {
		line, err := inputReader.ReadString('\n')
		line = strings.TrimRight(string(line), "\r\n")
		outs = append(outs, line)
		if err != nil {
			break
		}
	}
	_ = fd.Close()
	return outs, nil
}

func ReadFile(fileName string, maxLen uint) (string, error) {
	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0400)
	if err != nil {
		return "", err
	}

	buf := make([]byte, maxLen)
	n, err := fd.Read(buf)
	_ = fd.Close()
	if err != nil {
		return "", err
	}

	return string(buf[0:n]), nil
}

func WriteFile(fileName string, content string) error {
	CheckPath(fileName)

	fd, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	_, err = fd.Write([]byte(content))
	_ = fd.Close()
	if err != nil {
		return err
	}

	return nil
}

func FileExists(fileName string) bool {
	fi, err := os.Stat(fileName)
	return err == nil && fi != nil
}

func CheckPath(fileName string) {
	pos := strings.LastIndexByte(fileName, os.PathSeparator)
	if pos < 0 {
		return
	}
	path := fileName[0:pos]
	if _, err := os.Stat(path); err != nil {
		_ = os.MkdirAll(path, 0700)
	}
}

func FixPath(path string) string {
	const spe = string(os.PathSeparator)
	if !strings.HasSuffix(path, spe) {
		return path + spe
	}
	return path
}

var fileLocksLock = sync.Mutex{}
var fileLocks = map[string]*sync.Mutex{}

func Load(fileName string, to interface{}) error {
	if strings.HasSuffix(fileName, "yml") || strings.HasSuffix(fileName, "yaml") {
		return load(fileName, true, to)
	} else {
		return load(fileName, false, to)
	}
}

func LoadYaml(fileName string, to interface{}) error {
	return load(fileName, true, to)
}

func LoadJson(fileName string, to interface{}) error {
	return load(fileName, false, to)
}

func load(fileName string, isYaml bool, to interface{}) error {
	fileLocksLock.Lock()
	if fileLocks[fileName] == nil {
		fileLocks[fileName] = new(sync.Mutex)
	}
	lock := fileLocks[fileName]
	fileLocksLock.Unlock()

	lock.Lock()
	defer lock.Unlock()

	fp, err := os.Open(fileName)
	if err == nil {
		if isYaml {
			decoder := yaml.NewDecoder(fp)
			err = decoder.Decode(to)
		} else {
			decoder := json.NewDecoder(fp)
			err = decoder.Decode(to)
		}
		_ = fp.Close()
	}
	return err
}

func Save(fileName string, data interface{}) error {
	if strings.HasSuffix(fileName, "yml") || strings.HasSuffix(fileName, "yaml") {
		return save(fileName, true, data, true)
	} else {
		return save(fileName, false, data, false)
	}
}

func SaveYaml(fileName string, data interface{}) error {
	return save(fileName, true, data, true)
}

func SaveJson(fileName string, data interface{}) error {
	return save(fileName, false, data, false)
}

func SaveJsonP(fileName string, data interface{}) error {
	return save(fileName, false, data, true)
}

func save(fileName string, isYaml bool, data interface{}, indent bool) error {
	CheckPath(fileName)

	fileLocksLock.Lock()
	if fileLocks[fileName] == nil {
		fileLocks[fileName] = new(sync.Mutex)
	}
	lock := fileLocks[fileName]
	fileLocksLock.Unlock()

	lock.Lock()
	defer lock.Unlock()

	fp, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err == nil {
		var buf []byte
		if isYaml {
			buf, err = yaml.Marshal(data)
		} else {
			buffer := bytes.Buffer{}
			enc := json.NewEncoder(&buffer)
			enc.SetEscapeHTML(false)
			if indent {
				enc.SetIndent("", "  ")
			}
			err := enc.Encode(data)

			//buf, err = json.Marshal(data)
			if err == nil {
				buf = buffer.Bytes()
				FixUpperCase(buf, nil)
				//if indent {
				//	buf2 := bytes.Buffer{}
				//	err2 := json.Indent(&buf2, buf, "", "  ")
				//	if err2 == nil {
				//		buf = buf2.Bytes()
				//	}
				//}
			}
		}
		if err == nil {
			_, err = fp.Write(buf)
			_ = fp.Close()
		}
	}
	return err
}
