// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	readline "github.com/knz/go-libedit"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/server/serverpb"
	"github.com/cockroachdb/cockroach/pkg/util/envutil"
	"github.com/cockroachdb/cockroach/pkg/util/log"
)

const (
	debugInfoMessage = `# Welcome to the cockroach debug interface.
# For help, try: \?, \h, or \h [NAME].
# To exit: CTRL + D or \q.
#
`
)

func printDebugCliHelp() {
	fmt.Printf(`You are using 'cockroach debug shell', CockroachDB's lightweight debug client.
Type:
  \q        exit the shell (Ctrl+C/Ctrl+D also supported)
  \! CMD    run an external command and print its results on standard output.
  \| CMD    run an external command and run its output as debug statements.
  \?        print this help.
  \h [NAME] help on a debug command.

More documentation on the debug shell is available online:
%s
`,
		base.DocsURL("one-day-simba.html"),
	)
	fmt.Println()
}

var debugShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "open a debug shell",
	Long: `
Open a debug shell against a cockroach database.
`,
	Args: cobra.NoArgs,
	RunE: MaybeDecorateGRPCError(runShell),
}

func runShell(cmd *cobra.Command, args []string) error {
	checkInteractive()

	if cliCtx.isInteractive {
		fmt.Print(debugInfoMessage)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, finish, err := getClientGRPCConn(ctx)
	if err != nil {
		return err
	}
	defer finish()

	return runDebugInteractive(conn)
}

type shellStateEnum int

const (
	shellStart shellStateEnum = iota
	shellStop
	shellStartLine
	shellRefreshPrompt
	shellReadLine
	shellHandleCliCmd
	shellPrepareCmd
	shellRunCmd
)

var debugHistFile = envutil.EnvOrDefaultString("COCKROACH_DEBUG_CLI_HISTORY",
	".cockroachdebug_history")

type shellState struct {
	// Components
	conn *grpc.ClientConn
	ins  readline.EditLine
	buf  *bufio.Reader

	// Options
	errExit bool

	// State
	exitErr       error
	lastInputLine string
}

func (s *shellState) hasEditor() bool {
	return s.ins != noLineEditor
}

func (s *shellState) addHistory(line string) {
	if !s.hasEditor() || len(line) == 0 {
		return
	}

	if err := s.ins.AddHistory(line); err != nil {
		log.Warningf(context.TODO(), "cannot save command-line history: %s", err)
		log.Infof(context.TODO(), "command-line history will not be saved in this session")
		s.ins.SetAutoSaveHistory("", false)
	}
}

func (s *shellState) GetCompletions(needle string) []string {
	results := make([]string, 0)
	if strings.HasPrefix("problemranges", needle) {
		results = append(results, "problemranges")
	}
	if strings.HasPrefix("range", needle) {
		results = append(results, "range")
	}

	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 {
		return results
	}

	fmt.Fprint(stderr, "\n")
	for res := range results {
		fmt.Fprintf(stderr, "%v ", results[res])
	}
	fmt.Fprint(stderr, "\n\n")

	return nil
}

func (s *shellState) doStart(nextState shellStateEnum) shellStateEnum {
	if cliCtx.isInteractive {
		s.errExit = false
	} else {
		s.errExit = true
	}

	if s.hasEditor() {
		s.ins.SetCompleter(s)
		if err := s.ins.UseHistory(
			-1,   /* maxEntries */
			true, /* dedupe */
		); err != nil {
			log.Warningf(context.TODO(), "cannot enable history: %v", err)
		} else {
			homeDir, err := envutil.HomeDir()
			if err != nil {
				log.Warningf(context.TODO(), "cannot retrieve user information: %v", err)
				log.Infof(context.TODO(), "command-line history will not be saved in this session")
			} else {
				histFile := filepath.Join(homeDir, debugHistFile)
				err = s.ins.LoadHistory(histFile)
				if err != nil {
					log.Warningf(context.TODO(), "cannot load the command-line history file (file corrupted?): %v", err)
					log.Warningf(context.TODO(), "the history file will be cleared upon first entry")
				}
				s.ins.SetAutoSaveHistory(histFile, true)
			}
		}
	}

	return nextState
}

func (s *shellState) doStartLine(nextState shellStateEnum) shellStateEnum {
	return nextState
}

func (s *shellState) doRefreshPrompt(nextState shellStateEnum) shellStateEnum {
	if !s.hasEditor() {
		return nextState
	}

	s.ins.SetLeftPrompt("> ")

	return nextState
}

func (s *shellState) doReadLine(nextState shellStateEnum) shellStateEnum {
	l, err := s.ins.GetLine()
	if len(l) > 0 && l[len(l)-1] == '\n' {
		l = l[:len(l)-1]
	} else {
		fmt.Fprintln(s.ins.Stdout())
	}

	switch err {
	case nil:

	case readline.ErrInterrupted:
		if !cliCtx.isInteractive {
			s.exitErr = err
			return shellStop
		}

		if l != "" {
			return shellReadLine
		}

		s.exitErr = err
		return shellStop

	default:
		fmt.Fprintf(stderr, "input error: %s\n", err)
		s.exitErr = err
		return shellStop
	}

	s.lastInputLine = l
	return nextState
}

func (s *shellState) runSyscmd(line string, nextState, errState shellStateEnum) shellStateEnum {
	command := strings.Trim(line, " \r\n\t\f")
	if command == "" {
		fmt.Fprintf(stderr, "Usage:\n  \\! [command]\n")
		s.exitErr = errInvalidSyntax
		return errState
	}

	cmdOut, err := execSyscmd(command)
	if err != nil {
		fmt.Fprintf(stderr, "command failed: %s\n", err)
		s.exitErr = err
		return errState
	}

	fmt.Print(cmdOut)
	return nextState
}

func (s *shellState) pipeSyscmd(line string, nextState, errState shellStateEnum) shellStateEnum {
	command := strings.Trim(line, " \r\n\t\f")
	if command == "" {
		fmt.Fprintf(stderr, "Usage:\n  \\| [command]\n")
		s.exitErr = errInvalidSyntax
		return errState
	}

	cmdOut, err := execSyscmd(command)
	if err != nil {
		fmt.Fprintf(stderr, "command failed: %s\n", err)
		s.exitErr = err
		return errState
	}

	result := strings.Trim(cmdOut, " \r\n\t\f")

	s.lastInputLine = result
	return nextState
}

func (s *shellState) handleHelp(line string, nextState, errState shellStateEnum) shellStateEnum {
	line = strings.Trim(line, " \r\n\t\f")
	cmd := strings.Fields(line)

	if len(cmd) == 0 {
		fmt.Println("Available commands:")
		fmt.Println()
		fmt.Println("  problemranges")
		fmt.Println("  range")
		fmt.Println()
		fmt.Println("Use \\h [NAME] for more details about a particular command, or")
		fmt.Println("try \\? for help with shell features.")
		return nextState
	}

	switch cmd[0] {
	case "problemranges":
		fmt.Println("Usage: problemranges")
		fmt.Println()
		fmt.Println("Load the problem ranges report to investigate underreplicated")
		fmt.Println("and unavailable ranges and other problems.")
		return nextState

	case "range":
		fmt.Println("Usage: range <range_id>")
		fmt.Println()
		fmt.Println("View all status details about a range.")
		return nextState

	}

	fmt.Println("No help available for: %v (is that a command?)", cmd[0])
	return errState
}

func (s *shellState) invalidSyntax(
	nextState shellStateEnum, format string, args ...interface{},
) shellStateEnum {
	fmt.Fprint(stderr, "invalid syntax: ")
	fmt.Fprintf(stderr, format, args...)
	fmt.Fprintln(stderr)
	s.exitErr = errInvalidSyntax
	return nextState
}

func (s *shellState) doHandleCliCmd(loopState, nextState shellStateEnum) shellStateEnum {
	line := s.lastInputLine
	if len(line) == 0 || line[0] != '\\' {
		return nextState
	}

	errState := loopState
	if s.errExit {
		errState = shellStop
	}

	s.addHistory(line)

	cmd := strings.Fields(line)
	switch cmd[0] {
	case `\q`, `\quit`, `\exit`:
		return shellStop

	case `\`, `\?`, `\help`:
		printDebugCliHelp()

	case `\!`:
		return s.runSyscmd(line[2:], loopState, errState)

	case `\|`:
		return s.pipeSyscmd(line[2:], nextState, errState)

	case `\h`:
		// TODO: this slice ----v in sql starts at 1, is that right?
		return s.handleHelp(line[2:], loopState, errState)

	default:
		return s.invalidSyntax(errState, `%s. Try \? for help.`, line)
	}

	return loopState
}

func (s *shellState) doPrepareCmd(startState, runState shellStateEnum) shellStateEnum {
	line := strings.Trim(s.lastInputLine, " \r\n\t\f")
	if line == "" {
		return startState
	}

	s.addHistory(s.lastInputLine)

	return runState
}

func (s *shellState) doRunCmd(startState shellStateEnum) shellStateEnum {
	cmd := strings.Fields(s.lastInputLine)
	switch cmd[0] {
	case "problemranges":
		s.runProblemRanges(cmd[1:])

	case "range":
		s.runRange(cmd[1:])

	default:
		fmt.Fprintf(stderr, "Unknown command: %v\n", cmd[0])
	}

	if s.exitErr != nil {
		fmt.Fprintf(stderr, "Error: %s\n", s.exitErr)

		if s.errExit {
			return shellStop
		} else {
			s.exitErr = nil
		}
	}

	return startState
}

func (s *shellState) runProblemRanges(args []string) {
	status := serverpb.NewStatusClient(s.conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if problems, err := status.ProblemRanges(ctx, &serverpb.ProblemRangesRequest{}); err != nil {
		s.exitErr = err
	} else {
		fmt.Printf("Problem Ranges:\n%#v\n", problems)
	}
}

func (s *shellState) runRange(args []string) {

	if len(args) != 1 {
		s.invalidSyntax(shellStop, "%s.  Try: range [RANGE_ID]", s.lastInputLine)
		return
	}

	rangeId, err := parseRangeID(args[0])
	if err != nil {
		s.exitErr = err
		return
	}

	status := serverpb.NewStatusClient(s.conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if report, err := status.Range(ctx, &serverpb.RangeRequest{RangeId: int64(rangeId)}); err != nil {
		s.exitErr = err
	} else {
		fmt.Printf("Range %v:\n%#v\n", rangeId, report)
	}
}

func runDebugInteractive(conn *grpc.ClientConn) error {
	s := shellState{conn: conn}
	state := shellStart

	for {
		if state == shellStop {
			break
		}
		switch state {
		case shellStart:
			// Needs to be here due to the defer.
			if cliCtx.isInteractive && cliCtx.terminalOutput {
				s.ins, s.exitErr = readline.InitFiles("cockroach",
					true, /* wideChars */
					stdin, os.Stdout, stderr)
				if s.exitErr == readline.ErrWidecharNotSupported {
					log.Warning(context.TODO(), "wide character support disabled")
					s.ins, s.exitErr = readline.InitFiles("cockroach",
						false, stdin, os.Stdout, stderr)
				}
				if s.exitErr != nil {
					return s.exitErr
				}
				s.ins.RebindControlKeys()
				defer s.ins.Close()
			} else {
				s.ins = noLineEditor
				s.buf = bufio.NewReader(stdin)
			}

			state = s.doStart(shellStartLine)

		case shellStartLine:
			state = s.doStartLine(shellRefreshPrompt)

		case shellRefreshPrompt:
			state = s.doRefreshPrompt(shellReadLine)

		case shellReadLine:
			state = s.doReadLine(shellHandleCliCmd)

		case shellHandleCliCmd:
			state = s.doHandleCliCmd(shellRefreshPrompt, shellPrepareCmd)

		case shellPrepareCmd:
			state = s.doPrepareCmd(shellStartLine, shellRunCmd)

		case shellRunCmd:
			state = s.doRunCmd(shellStartLine)
		}
	}

	return s.exitErr
}
