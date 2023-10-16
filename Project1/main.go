package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func main() {
	// CLI args
	f, closeFile, err := openProcessingFile(os.Args...)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile()

	// Load and parse processes
	processes, err := loadProcesses(f)
	if err != nil {
		log.Fatal(err)
	}

	// First-come, first-serve scheduling
	FCFSSchedule(os.Stdout, "First-come, first-serve", processes)

	SJFSchedule(os.Stdout, "Shortest-job-first", processes)
	//
	SJFPrioritySchedule(os.Stdout, "Priority", processes)
	//
	RRSchedule(os.Stdout, "Round-robin", processes, 2)
}

func openProcessingFile(args ...string) (*os.File, func(), error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%w: must give a scheduling file to process", ErrInvalidArgs)
	}
	// Read in CSV process CSV file
	f, err := os.Open(args[1])
	if err != nil {
		return nil, nil, fmt.Errorf("%v: error opening scheduling file", err)
	}
	closeFn := func() {
		if err := f.Close(); err != nil {
			log.Fatalf("%v: error closing scheduling file", err)
		}
	}

	return f, closeFn, nil
}

type (
	Process struct {
		ProcessID     int64
		ArrivalTime   int64
		BurstDuration int64
		Priority      int64
	}
	TimeSlice struct {
		PID   int64
		Start int64
		Stop  int64
	}
)

//region Schedulers

// FCFSSchedule outputs a schedule of processes in a GANTT chart and a table of timing given:
// • an output writer
// • a title for the chart
// • a slice of processes
func FCFSSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)
	for i := range processes {
		if processes[i].ArrivalTime > 0 {
			waitingTime = serviceTime - processes[i].ArrivalTime
		}
		totalWait += float64(waitingTime)

		start := waitingTime + processes[i].ArrivalTime

		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		completion := processes[i].BurstDuration + processes[i].ArrivalTime + waitingTime
		lastCompletion = float64(completion)

		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}
		serviceTime += processes[i].BurstDuration

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := totalWait / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

func SJFPrioritySchedule(w io.Writer, title string, processes []Process) {
	sort.Slice(processes, func(i, j int) bool {
		if processes[i].ArrivalTime != processes[j].ArrivalTime {
			return processes[i].ArrivalTime < processes[j].ArrivalTime
		}
		if processes[i].Priority != processes[j].Priority {
			return processes[i].Priority < processes[j].Priority
		}
		return processes[i].BurstDuration < processes[j].BurstDuration
	})

	var (
		currTime int64
		waitTime float64
		turn     float64
		gantt    = make([]TimeSlice, 0)
		schedule = make([][]string, len(processes))
	)

	remainingProcesses := make([]Process, len(processes))
	copy(remainingProcesses, processes)

	for len(remainingProcesses) > 0 {
		var shortest *Process
		for i := range remainingProcesses {
			if remainingProcesses[i].ArrivalTime <= currTime {
				if shortest == nil || remainingProcesses[i].BurstDuration < shortest.BurstDuration || (remainingProcesses[i].BurstDuration == shortest.BurstDuration && remainingProcesses[i].Priority < shortest.Priority) {
					shortest = &remainingProcesses[i]
				}
			}
		}

		if shortest == nil {
			nextArrivalTime := remainingProcesses[0].ArrivalTime
			for _, process := range remainingProcesses {
				if process.ArrivalTime > currTime && process.ArrivalTime < nextArrivalTime {
					nextArrivalTime = process.ArrivalTime
				}
			}
			currTime = nextArrivalTime
		} else {
			gantt = append(gantt, TimeSlice{
				PID:   shortest.ProcessID,
				Start: currTime,
				Stop:  currTime + shortest.BurstDuration,
			})

			waitTime += float64(currTime - shortest.ArrivalTime)
			turn += float64(currTime+shortest.BurstDuration-shortest.ArrivalTime) - waitTime

			schedule = append(schedule, []string{
				strconv.FormatInt(shortest.ProcessID, 10),
				strconv.FormatInt(shortest.Priority, 10),
				strconv.FormatInt(shortest.BurstDuration, 10),
				strconv.FormatInt(shortest.ArrivalTime, 10),
				strconv.FormatFloat(float64(currTime-shortest.ArrivalTime), 'f', 2, 64),
				strconv.FormatFloat(float64(currTime+shortest.BurstDuration-shortest.ArrivalTime), 'f', 2, 64),
				strconv.FormatInt(currTime+shortest.BurstDuration, 10),
			})

			for i, process := range remainingProcesses {
				if process.ProcessID == shortest.ProcessID {
					remainingProcesses = append(remainingProcesses[:i], remainingProcesses[i+1:]...)
					break
				}
			}

			currTime += shortest.BurstDuration
		}
	}

	throughput := float64(len(processes)) / float64(currTime)

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, waitTime/float64(len(processes)), turn/float64(len(processes)), throughput)
}

func SJFSchedule(w io.Writer, title string, processes []Process) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].ArrivalTime < processes[j].ArrivalTime
	})

	var (
		currTime int64
		waitTime float64
		turn     float64
		gantt    = make([]TimeSlice, 0)
		schedule = make([][]string, len(processes))
	)

	remainingProcesses := make([]Process, len(processes))
	copy(remainingProcesses, processes)

	for len(remainingProcesses) > 0 {
		shortestIndex := 0
		for i := range remainingProcesses {
			if remainingProcesses[i].BurstDuration < remainingProcesses[shortestIndex].BurstDuration {
				shortestIndex = i
			}
		}
		shortest := &remainingProcesses[shortestIndex]

		if shortest.ArrivalTime > currTime {
			currTime = shortest.ArrivalTime
		}

		gantt = append(gantt, TimeSlice{
			PID:   shortest.ProcessID,
			Start: currTime,
			Stop:  currTime + shortest.BurstDuration,
		})

		waitTime += float64(currTime - shortest.ArrivalTime)
		turn += float64(currTime+shortest.BurstDuration-shortest.ArrivalTime) - waitTime

		schedule = append(schedule, []string{
			strconv.FormatInt(shortest.ProcessID, 10),
			strconv.FormatInt(shortest.Priority, 10),
			strconv.FormatInt(shortest.BurstDuration, 10),
			strconv.FormatInt(shortest.ArrivalTime, 10),
			strconv.FormatFloat(float64(currTime-shortest.ArrivalTime), 'f', 2, 64),
			strconv.FormatFloat(float64(currTime+shortest.BurstDuration-shortest.ArrivalTime), 'f', 2, 64),
			strconv.FormatInt(currTime+shortest.BurstDuration, 10),
		})

		remainingProcesses = append(remainingProcesses[:shortestIndex], remainingProcesses[shortestIndex+1:]...)

		currTime += shortest.BurstDuration
	}

	throughput := float64(len(processes)) / float64(currTime)

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, waitTime/float64(len(processes)), turn/float64(len(processes)), throughput)
}

// quantum time
func RRSchedule(w io.Writer, title string, processes []Process, quantum int64) {
	outputTitle(w, title)

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].ArrivalTime < processes[j].ArrivalTime
	})

	var (
		gantt    = make([]TimeSlice, 0)
		schedule = make([][]string, len(processes))
		wait     float64
		turn     float64
		elapsed  int64
	)

	queue := make([]Process, len(processes))
	copy(queue, processes)

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		gantt = append(gantt, TimeSlice{p.ProcessID, elapsed, elapsed + min(p.BurstDuration, quantum)})
		elapsed += min(p.BurstDuration, quantum)

		p.BurstDuration -= min(p.BurstDuration, quantum)

		if p.BurstDuration > 0 {
			queue = append(queue, p)
		} else {
			w := float64(elapsed - p.ArrivalTime - priorityPenalty(p))
			t := float64(elapsed - p.ArrivalTime)
			wait += w
			turn += t

			schedule = append(schedule, []string{
				strconv.FormatInt(p.ProcessID, 10),
				strconv.FormatInt(p.Priority, 10),
				strconv.FormatInt(p.BurstDuration, 10),
				strconv.FormatInt(p.ArrivalTime, 10),
				fmt.Sprintf("%.2f", w),
				fmt.Sprintf("%.2f", t),
				strconv.FormatInt(elapsed, 10),
			})
		}
	}

	tp := float64(len(processes)) / float64(elapsed)

	outputGantt(w, gantt)
	outputSchedule(w, schedule, wait/float64(len(processes)), turn/float64(len(processes)), tp)
}

func priorityPenalty(p Process) int64 {
	return (p.Priority - 1) * 5
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func outputTitle(w io.Writer, title string) {
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
	_, _ = fmt.Fprintln(w, strings.Repeat(" ", len(title)/2), title)
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
}

func outputGantt(w io.Writer, gantt []TimeSlice) {
	_, _ = fmt.Fprintln(w, "Gantt schedule")
	_, _ = fmt.Fprint(w, "|")
	for i := range gantt {
		pid := fmt.Sprint(gantt[i].PID)
		padding := strings.Repeat(" ", (8-len(pid))/2)
		_, _ = fmt.Fprint(w, padding, pid, padding, "|")
	}
	_, _ = fmt.Fprintln(w)
	for i := range gantt {
		_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Start), "\t")
		if len(gantt)-1 == i {
			_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Stop))
		}
	}
	_, _ = fmt.Fprintf(w, "\n\n")
}

func outputSchedule(w io.Writer, rows [][]string, wait, turnaround, throughput float64) {
	_, _ = fmt.Fprintln(w, "Schedule table")
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"ID", "Priority", "Burst", "Arrival", "Wait", "Turnaround", "Exit"})
	table.AppendBulk(rows)
	table.SetFooter([]string{"", "", "", "",
		fmt.Sprintf("Average\n%.2f", wait),
		fmt.Sprintf("Average\n%.2f", turnaround),
		fmt.Sprintf("Throughput\n%.2f/t", throughput)})
	table.Render()
}

//endregion

//region Loading processes.

var ErrInvalidArgs = errors.New("invalid args")

func loadProcesses(r io.Reader) ([]Process, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: reading CSV", err)
	}

	processes := make([]Process, len(rows))
	for i := range rows {
		processes[i].ProcessID = mustStrToInt(rows[i][0])
		processes[i].BurstDuration = mustStrToInt(rows[i][1])
		processes[i].ArrivalTime = mustStrToInt(rows[i][2])
		if len(rows[i]) == 4 {
			processes[i].Priority = mustStrToInt(rows[i][3])
		}
	}

	return processes, nil
}

func mustStrToInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return i
}

//endregion

