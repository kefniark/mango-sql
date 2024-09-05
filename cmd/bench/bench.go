package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func parseBenchmarks() []BenchData {
	f, err := os.OpenFile("bench.log", os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	entries := []BenchData{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		fields := strings.Fields(line)

		// name
		suite := strings.ReplaceAll(strings.Split(fields[0], "/")[0], "Benchmark", "")
		sub := strings.Split(fields[0], "/")[1]
		name := sub
		param := "1"
		if strings.Contains(sub, "_") {
			name = strings.Split(strings.Split(fields[0], "/")[1], "_")[0]
			param = strings.Split(strings.Split(fields[0], "/")[1], "_")[1]

			if strings.Contains(param, "-") {
				param = strings.Split(param, "-")[0]
			}
		}

		// math
		paramValue, _ := strconv.Atoi(param)
		iteration, _ := strconv.Atoi(fields[1])
		speed, _ := strconv.Atoi(fields[2])
		data, _ := strconv.Atoi(fields[4])
		alloc, _ := strconv.Atoi(fields[6])

		entries = append(entries, BenchData{
			Suite: suite,
			Name:  name,
			Param: paramValue,
			Iter:  iteration,
			Speed: speed,
			Data:  data,
			Alloc: alloc,
		})
	}
	return entries
}

type BenchSeries struct {
	Name string
	Data []opts.LineData
}

type BenchData struct {
	Suite string
	Name  string
	Param int
	Iter  int
	Speed int
	Data  int
	Alloc int
}

func groupBySuite(data []BenchData) map[string][]BenchData {
	entries := map[string][]BenchData{}
	for _, bench := range data {
		entries[bench.Suite] = append(entries[bench.Suite], bench)
	}
	return entries
}

const unitConversion = 1000000000

func generateLineSpeedData(data []BenchData, filters ...string) []opts.LineData {
	items := make([]opts.LineData, 0)
	for _, v := range data {
		for _, f := range filters {
			if !strings.Contains(v.Name, f) {
				continue
			}
		}
		items = append(items, opts.LineData{Value: unitConversion / v.Speed})
	}
	return items
}

func generateLineAllocData(data []BenchData, filters ...string) []opts.LineData {
	items := make([]opts.LineData, 0)
	for _, v := range data {
		for _, f := range filters {
			if !strings.Contains(v.Name, f) {
				continue
			}
		}
		items = append(items, opts.LineData{Value: v.Alloc})
	}
	return items
}

func generateInsertManyCPULines(data []BenchData, name string, filter string) *charts.Line {
	entries := groupBySuite(data)

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    name + "|CPU",
			Subtitle: "Higher is Better (op/s)",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "shine",
			Width:  "576px",
			Height: "320px",
		}),
	)
	line.SetXAxis([]string{"10", "25", "50", "100", "250", "500"})

	seriesData := []BenchSeries{}
	for k, v := range entries {
		if !strings.Contains(k, filter) {
			continue
		}
		seriesData = append(seriesData, BenchSeries{
			Name: strings.ReplaceAll(k, filter, ""),
			Data: generateLineSpeedData(v, name),
		})
	}

	drawLines(seriesData, line)

	return line
}

func generateInsertManyAllocLines(data []BenchData, name string, filter string) *charts.Line {
	entries := groupBySuite(data)

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    name + "|Mem",
			Subtitle: "Lower is Better (allocs/op)",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "shine",
			Width:  "576px",
			Height: "320px",
		}),
	)
	line.SetXAxis([]string{"10", "25", "50", "100", "250", "500"})

	seriesData := []BenchSeries{}
	for k, v := range entries {
		if !strings.Contains(k, filter) {
			continue
		}
		seriesData = append(seriesData, BenchSeries{
			Name: strings.ReplaceAll(k, filter, ""),
			Data: generateLineAllocData(v, name),
		})
	}

	drawLines(seriesData, line)

	return line
}

func drawLines(seriesData []BenchSeries, line *charts.Line) {
	slices.SortFunc(seriesData, func(a, b BenchSeries) int {
		return strings.Compare(a.Name, b.Name)
	})

	for _, v := range seriesData {
		line.AddSeries(v.Name, v.Data)
	}
}

func main() {
	data := parseBenchmarks()

	for _, db := range []string{"Postgres", "SQLite", "MariaDB"} {
		for _, op := range []string{"InsertMany", "FindMany"} {
			f1, _ := os.Create(fmt.Sprintf("docs/public/bench_%s_%s_cpu.html", strings.ToLower(db), strings.ToLower(op)))
			defer f1.Close()
			err := generateInsertManyCPULines(data, op, db).Render(f1)
			if err != nil {
				panic(err)
			}

			f2, _ := os.Create(fmt.Sprintf("docs/public/bench_%s_%s_alloc.html", strings.ToLower(db), strings.ToLower(op)))
			defer f2.Close()
			err = generateInsertManyAllocLines(data, op, db).Render(f2)
			if err != nil {
				panic(err)
			}
		}
	}
}
