package procstat

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/process"

	"github.com/influxdb/telegraf/plugins/inputs"
)

type Procstat struct {
	PidFile string `toml:"pid_file"`
	Exe     string
	Pattern string
	Prefix  string

	pidmap map[int32]*process.Process
}

func NewProcstat() *Procstat {
	return &Procstat{
		pidmap: make(map[int32]*process.Process),
	}
}

var sampleConfig = `
  # Must specify one of: pid_file, exe, or pattern
  # PID file to monitor process
  pid_file = "/var/run/nginx.pid"
  # executable name (ie, pgrep <exe>)
  # exe = "nginx"
  # pattern as argument for pgrep (ie, pgrep -f <pattern>)
  # pattern = "nginx"

  # Field name prefix
  prefix = ""
`

func (_ *Procstat) SampleConfig() string {
	return sampleConfig
}

func (_ *Procstat) Description() string {
	return "Monitor process cpu and memory usage"
}

func (p *Procstat) Gather(acc inputs.Accumulator) error {
	err := p.createProcesses()
	if err != nil {
		log.Printf("Error: procstat getting process, exe: [%s] pidfile: [%s] pattern: [%s] %s",
			p.Exe, p.PidFile, p.Pattern, err.Error())
	} else {
		for _, proc := range p.pidmap {
			p := NewSpecProcessor(p.Prefix, acc, proc)
			p.pushMetrics()
		}
	}

	return nil
}

func (p *Procstat) createProcesses() error {
	var errstring string
	var outerr error

	pids, err := p.getAllPids()
	if err != nil {
		errstring += err.Error() + " "
	}

	for _, pid := range pids {
		_, ok := p.pidmap[pid]
		if !ok {
			proc, err := process.NewProcess(pid)
			if err == nil {
				p.pidmap[pid] = proc
			} else {
				errstring += err.Error() + " "
			}
		}
	}

	if errstring != "" {
		outerr = fmt.Errorf("%s", errstring)
	}

	return outerr
}

func (p *Procstat) getAllPids() ([]int32, error) {
	var pids []int32
	var err error

	if p.PidFile != "" {
		pids, err = pidsFromFile(p.PidFile)
	} else if p.Exe != "" {
		pids, err = pidsFromExe(p.Exe)
	} else if p.Pattern != "" {
		pids, err = pidsFromPattern(p.Pattern)
	} else {
		err = fmt.Errorf("Either exe, pid_file or pattern has to be specified")
	}

	return pids, err
}

func pidsFromFile(file string) ([]int32, error) {
	var out []int32
	var outerr error
	pidString, err := ioutil.ReadFile(file)
	if err != nil {
		outerr = fmt.Errorf("Failed to read pidfile '%s'. Error: '%s'", file, err)
	} else {
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidString)))
		if err != nil {
			outerr = err
		} else {
			out = append(out, int32(pid))
		}
	}
	return out, outerr
}

func pidsFromExe(exe string) ([]int32, error) {
	var out []int32
	var outerr error
	pgrep, err := exec.Command("pgrep", exe).Output()
	if err != nil {
		return out, fmt.Errorf("Failed to execute pgrep. Error: '%s'", err)
	} else {
		pids := strings.Fields(string(pgrep))
		for _, pid := range pids {
			ipid, err := strconv.Atoi(pid)
			if err == nil {
				out = append(out, int32(ipid))
			} else {
				outerr = err
			}
		}
	}
	return out, outerr
}

func pidsFromPattern(pattern string) ([]int32, error) {
	var out []int32
	var outerr error
	pgrep, err := exec.Command("pgrep", "-f", pattern).Output()
	if err != nil {
		return out, fmt.Errorf("Failed to execute pgrep. Error: '%s'", err)
	} else {
		pids := strings.Fields(string(pgrep))
		for _, pid := range pids {
			ipid, err := strconv.Atoi(pid)
			if err == nil {
				out = append(out, int32(ipid))
			} else {
				outerr = err
			}
		}
	}
	return out, outerr
}

func init() {
	inputs.Add("procstat", func() inputs.Input {
		return NewProcstat()
	})
}
