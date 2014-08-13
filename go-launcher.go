package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strings"
)

var (
	appPath = flag.String("appPath", "/path/to/go/service", "Path to executable app")
	config  = flag.String("config", "dev.cfg", "Path to config with command-line flag values. The path may be relative to appPath")
)

var (
	LINES_REGEXP = regexp.MustCompile("[\\r\\n]")
	KV_REGEXP    = regexp.MustCompile("\\s*=\\s*")
)

func main() {
	flag.Parse()

	appDir := path.Dir(*appPath)
	configPath := *config
	if configPath[0] != '/' {
		configPath = path.Join(appDir, configPath)
	}
	args := getArgsFromConfig(configPath)
	cmd := exec.Command("./" + path.Base(*appPath), args...)
	cmd.Dir = appDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error when launching app=[%s] with config=[%s], args=[%s]: [%s]\n", *appPath, *config, strings.Join(args, " "), err)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan)
	go func() {
		for sig := range sigChan {
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		log.Fatalf("Error when waiting for the app=[%s] with config=[%s], args=[%s]: [%s]\n", *appPath, *config, strings.Join(args, " "), err)
	}
}

func getArgsFromConfig(configPath string) []string {
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("cannot open config file at [%s]: [%s]\n", configPath, err)
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Error when reading config file [%s]: [%s]\n", configPath, err)
	}

	var args []string
	for _, line := range LINES_REGEXP.Split(string(data), -1) {
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		parts := KV_REGEXP.Split(line, 2)
		if len(parts) != 2 {
			log.Fatalf("Cannot split line=[%s] into key and value in config file [%s]", line, configPath)
		}
		key := parts[0]
		value := unquoteValue(parts[1])
		args = append(args, "-" + key, value)
	}
	return args
}

func unquoteValue(v string) string {
	if v[0] != '"' {
		return v
	}
	n := strings.LastIndex(v, "\"")
	if n == -1 {
		return v
	}
	v = v[1:n]
	v = strings.Replace(v, "\\\"", "\"", -1)
	return strings.Replace(v, "\\n", "\n", -1)
}
