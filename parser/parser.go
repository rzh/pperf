package parser

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// var re_process_line = regexp.MustCompile("([a-zA-Z0-9]+) ([0-9]+) ([0-9.]+): cpu-clock:")
// var re_frame_line = regexp.MustCompile("([a-z0-9]+) ([^ ]+) ([^ ]+)")
const topN int = 5
const topFunc bool = true

type Parser interface {
}

type Function struct {
	Function       string
	ExecutionSpace string // kernel or process
	Usage          float64
}

type PerfFrame struct {
	TS        float64
	Process   string     // name of the process
	Pid       int64      // pid of the process
	Functions []Function // list of function, from top of stack to bottom
}

type PerfFrames []PerfFrame

// parse file to get one frame
// input:
//     r *os.File - file to be read
//     s string   - the first line of the frame, passing in so no need to rewind
// return:
//     perfFrame
func parseOneFrame(r *bufio.Scanner, firstLine string) (PerfFrame, error) {
	// it is already a frame, parse to create a perfFrame struct
	var pf PerfFrame

	t := strings.Split(strings.TrimSpace(firstLine), " ")

	if len(t) > 0 {
		pf.Process = t[0]
		var i int64
		var f float64
		var err error

		if i, err = strconv.ParseInt(t[1], 10, 64); err != nil {
			log.Panicf("Failed to parse PID for line [%s]\n", firstLine)
		}
		pf.Pid = i

		if f, err = strconv.ParseFloat(t[2][:len(t[2])-1], 64); err != nil {
			return PerfFrame{}, errors.New(fmt.Sprintf("Failed to parse PID for line [%s]\n", firstLine))
		}
		pf.TS = f
	} else {
		// panic and quit
		pf.Process = "n/a"
	}

	// now process frame line by line
	for r.Scan() {
		s := strings.TrimSpace(r.Text())
		t := strings.Split(s, " ")

		// t := re_frame_line.FindStringSubmatch(strings.Replace(s, ", ", ",", -1))

		if len(t) > 1 {
			// process frame
			//fmt.Println("===", s, len(t))
			fName := strings.Join(t[1:len(t)-1], " ")
			pf.Functions = append(pf.Functions, Function{Function: fName, ExecutionSpace: t[len(t)-1]})
		} else {
			// frame end
			return pf, nil
		}
	}

	return pf, nil
}

func ParsePerfScript(f *os.File) (PerfFrames, error) {
	var frames PerfFrames

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		s := scanner.Text()

		if s != "" && s[0] != ' ' {
			// this is the first line of a frame
			f, err := parseOneFrame(scanner, s)

			if err != nil {
				// do something here
				return nil, err
			}

			frames = append(frames, f)
		}
	}

	return frames, nil
}

type TimeSlot struct {
	TS        float64
	F         []FuncCount
	NumSample int // how many samples in the timeslot
}

type FuncCount struct {
	Count      int64
	Percentage float64
	F          Function
}

type Timeline []TimeSlot

func calculateTopN(topFunctions map[string]int, currentTs float64, currentNumSample int) (times TimeSlot) {

	var topNFunction []string = make([]string, topN, topN)
	var topNCount [topN]int

	for k, v := range topFunctions {
		for i := 0; i < topN; i++ {
			// search bottom up
			if v > topNCount[i] {
				tk := topNFunction[i]
				tv := topNCount[i]
				topNCount[i] = v
				topNFunction[i] = k
				k = tk
				v = tv
			}
		}
	}

	times.F = make([]FuncCount, topN, topN)
	times.TS = currentTs
	times.NumSample = currentNumSample

	for i := 0; i < topN; i++ {
		times.F[i] = FuncCount{F: Function{Function: topNFunction[i]},
			Count:      int64(topNCount[i]),
			Percentage: 100 * float64(topNCount[i]) / float64(currentNumSample),
		}

	}

	return
}

// parse *os.File and produce a timeline
func ParsePerfScriptTimeline(f *os.File) (Timeline, error) {
	var timeline Timeline

	log.Println("parse file now")

	pf, err := ParsePerfScript(f)
	if err != nil {
		return nil, err
	}

	log.Println("parse file done")

	// find top N common used top function

	var currentTs float64
	var currentNumSample int
	var topFunctions map[string]int

	topFunctions = make(map[string]int)
	for _, frame := range pf {

		if currentTs == 0 {
			currentTs = math.Floor(frame.TS)
		}

		if math.Floor(frame.TS) == currentTs {
			// in the same second
			currentNumSample = currentNumSample + 1

			topFunctions[frame.Functions[0].Function] = topFunctions[frame.Functions[0].Function] + 1
			//fmt.Println("increase counter for ", frame.Functions[0].Function)
		} else {
			// a new timeslot
			// find topN for this frame

			timeline = append(timeline, calculateTopN(topFunctions, currentTs, currentNumSample))
			topFunctions = make(map[string]int)
			currentTs = 0
			currentNumSample = 0
		}
	}

	// we need add the last frame
	if currentTs > 0 {
		timeline = append(timeline, calculateTopN(topFunctions, currentTs, currentNumSample))
	}

	log.Println("done for all tasks")
	return timeline, nil
}

func PrintPerfTimeline(tl Timeline) {
	for _, t := range tl {
		fmt.Printf("\nTimestamp: %10.0f\n", math.Floor(t.TS))
		for i := 0; i < topN; i++ {
			if len(t.F[i].F.Function) <= 40 {
				fmt.Printf("    %40s :%4d [%5.2f%%]\n", t.F[i].F.Function, t.F[i].Count, t.F[i].Percentage)

			} else {
				fmt.Printf("    %40s :%4d [%5.2f%%]\n", t.F[i].F.Function[:35]+".....", t.F[i].Count, t.F[i].Percentage)
			}
		}
	}
}
