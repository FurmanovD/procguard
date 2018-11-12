package procguard

import (
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

//
// ProcessGuard describes a process guard structure.
//
type ProcessGuard struct {
	Command          string
	Args             []string
	Environment      map[string]string
	MaxRestartOnFail int
	RunStatistics    []RunStat //TODO

	cmd *exec.Cmd

	onceRun sync.Once
	wgDone  *sync.WaitGroup

	// to be able to stop all externally
	chStopListener chan bool
	chFinished     chan bool

	runTry int
}

//
// RunGuarded starts a new process in a "guarded" environment
//
func (t *ProcessGuard) RunGuarded() error {
	t.onceRun.Do(func() {
		t.runTry = -1
		t.wgDone.Add(1)
		go t.execListener()
	})

	return nil
}

//
// Stop waits for a process to finish
//
func (t *ProcessGuard) Stop() {
	t.chStopListener <- true
}

//
// startProcess starts a new process
//
func (t *ProcessGuard) startProcess() error {

	t.runTry++
	thisRun := RunStat{}
	t.RunStatistics = append(t.RunStatistics, thisRun)
	t.RunStatistics[t.runTry].Start = time.Now()
	t.cmd = exec.Command(t.Command, t.Args...)

	if 0 < len(t.Environment) {
		for k, v := range t.Environment {
			t.cmd.Env = append(t.cmd.Env, k+"="+v)
		}
	}

	if err := t.cmd.Start(); nil != err {
		t.RunStatistics[t.runTry].Error = err
		t.RunStatistics[t.runTry].Finish = time.Now()
		return err
	}
	go t.cmd.Wait()

	return nil
}

//
// StdoutPipe proxy method
//
func (t *ProcessGuard) StdoutPipe() (io.ReadCloser, error) {
	if nil == t || nil == t.cmd {
		return nil, errors.New("Start guarded process to call StdoutPipe")
	}
	return t.cmd.StdoutPipe()
}

//
// StderrPipe proxy method
//
func (t *ProcessGuard) StderrPipe() (io.ReadCloser, error) {
	if nil == t || nil == t.cmd {
		return nil, errors.New("Start guarded process to call StderrPipe")
	}
	return t.cmd.StderrPipe()
}

//
// execListener listens a process state and manages it
//
func (t *ProcessGuard) execListener() {

	t.startProcess()

	checkStateTimer := time.NewTimer(time.Millisecond * 500)
	go func() {
		<-checkStateTimer.C
		if nil != t.cmd.ProcessState {
			t.chFinished <- true
			//t.RunStatistics[t.runTry].Error = nil //TODO
			t.RunStatistics[t.runTry].Finish = time.Now()
		}
	}()

	continueLoop := true
	for continueLoop {
		select {
		case <-t.chStopListener:
			continueLoop = false
			log.Debugln("execListener got Stop signal")
			break
		case <-t.chFinished:
			//TODO check state and restart if required
			t.startProcess()

		}
	}
	t.wgDone.Done()
}
