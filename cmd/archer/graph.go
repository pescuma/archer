package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/utils"
)

type GraphCmd struct {
	cmdWithFilters

	Output string `short:"o" default:"deps.png" help:"Output file to write." type:"path"`
	Levels int    `short:"l" help:"How many levels of subprojects should be considered."`
	Lines  bool   `default:"true" negatable:"" help:"Scale nodes by the number of lines."`
}

func (c *GraphCmd) Run(ctx *context) error {
	projects, err := ctx.ws.LoadProjects()
	if err != nil {
		return err
	}

	filter, err := c.createFilter(projects)
	if err != nil {
		return err
	}

	dot := c.generateDot(projects, filter)

	gv := c.Output + ".gv"

	fmt.Printf("Creating dot file: %v\n", gv)

	err = os.WriteFile(gv, []byte(dot), 0o600)
	if err != nil {
		return err
	}

	format := filepath.Ext(c.Output)
	if format == "" {
		c.Output += ".png"
		format = "png"
	} else {
		format = format[1:]
	}

	fmt.Printf("Generating output graph: %v\n", c.Output)

	cmd := exec.Command("dot", gv, "-T"+format, "-o", c.Output)
	err = cmd.Run()
	if err != nil {
		return err
	}

	err = os.Remove(gv)
	if err != nil {
		return err
	}

	return nil
}

func (c *GraphCmd) generateDot(projects *archer.Projects, filter archer.Filter) string {
	ps := projects.ListProjects(archer.FilterExcludeExternal)

	getProjectName := func(p *archer.Project) string {
		result := p.LevelSimpleName(c.Levels)
		result = strings.TrimSuffix(result, "-api")
		return result
	}

	tg := groupByRoot(ps, filter, true, getProjectName)

	nodes := map[string]*node{}
	colors := c.computeColors(ps, getProjectName)
	showSizes, computeGraphSize := c.computeSizesConfig(tg)

	o := newOutput()
	o.addLine(`digraph G {`)

	for _, rg := range tg.children {
		o.addLine(`subgraph "cluster_%v" {`, rg.name)

		if showSizes && !rg.size.isEmpty() {
			o.addLine(`label = <%v<font point-size="9" color="dimgrey"><br/>%v</font>>`, rg.name, rg.size.html())
		} else {
			o.addLine(`label = "%v"`, rg.name)
		}

		for _, pg := range rg.children {
			n, ok := nodes[pg.fullName]
			if !ok {
				n = newNode(pg.fullName)
				n.attribs["color"] = colors[pg.fullName]

				nodes[pg.fullName] = n
			}

			if showSizes && !pg.size.isEmpty() {
				n.attribs["shape"] = "circle"
				n.attribs["fixedsize"] = "shape"
				n.attribs["width"] = humanize.FormatFloat("#.##", computeGraphSize(pg.size.get()))

				n.attribs["label"] = fmt.Sprintf(`<%v<font point-size="9" color="dimgrey"><br/>%v</font>>`,
					pg.name,
					pg.size.html(),
				)
			} else {
				n.attribs["label"] = pg.name
			}

			o.addLineDistinct(n)
		}

		o.addLine("}")
		o.addLine("")
	}

	for _, rg := range tg.children {
		for _, pg := range rg.children {
			for _, dg := range pg.children {
				e := newEdge(pg.fullName, dg.fullName)
				e.attribs["color"] = colors[dg.fullName]
				e.attribs["style"] = dg.dep.GetData("style")

				o.addLineDistinct(e)
			}
		}
	}
	o.addLine("")

	if showSizes && !tg.size.isEmpty() {
		o.addLine("{ rank = sink; legend_Total [shape=plaintext label=<Total<br/>%v>] }", tg.size.html())
	}

	o.addLine("}")

	return o.String()
}

func (c *GraphCmd) computeSizesConfig(tg *group) (bool, func(int) float64) {
	ls := []int{-1, -1}
	for _, rg := range tg.children {
		for _, pg := range rg.children {
			s := pg.size.get()
			if s > 0 {
				if ls[0] == -1 {
					ls[0] = s
					ls[1] = s
				} else {
					ls[0] = utils.Min(ls[0], s)
					ls[1] = utils.Max(ls[1], s)
				}
			}
		}
	}

	showSizes := c.Lines && ls[0] != -1

	sizeRange := []float64{0.5, 2}
	if showSizes && ls[1] > 10*ls[0] {
		sizeRange = []float64{0.1, 10.0}
	}

	return showSizes, func(size int) float64 {
		f := float64(size-ls[0]) / float64(ls[1])
		f = math.Sqrt(f)
		return utils.Max(f*(sizeRange[1]), sizeRange[0])
	}
}

func (c *GraphCmd) computeNodesShow(ps []*archer.Project, filter archer.Filter) map[string]bool {
	show := map[string]bool{}

	for _, p := range ps {
		show[p.Name] = false
	}

	for _, p := range ps {
		if !show[p.Name] {
			show[p.Name] = filter.Decide(filter.FilterProject(p)) != archer.Exclude
		}

		for _, d := range p.ListDependencies(archer.FilterExcludeExternal) {
			if filter.Decide(filter.FilterDependency(d)) == archer.Exclude {
				continue
			}

			show[d.Source.Name] = true
			show[d.Target.Name] = true
		}
	}

	return show
}

func (c *GraphCmd) computeColors(ps []*archer.Project, getProjectName func(p *archer.Project) string) map[string]string {
	availableColors := []string{
		"#1abc9c",
		"#16a085",
		"#2ecc71",
		"#27ae60",
		"#3498db",
		"#2980b9",
		"#9b59b6",
		"#8e44ad",
		"#34495e",
		"#2c3e50",
		// "#f1c40f",
		"#f39c12",
		"#e67e22",
		"#d35400",
		// "#e74c3c",
		// "#c0392b",
		// "#ecf0f1",
		// "#bdc3c7",
		"#95a5a6",
		"#7f8c8d",
	}
	aci := 0

	colors := map[string]string{}

	for _, p := range ps {
		pn := p.Root + ":" + getProjectName(p)

		color := p.GetData("color")
		if color != "" {
			colors[pn] = color

		} else {
			color = availableColors[aci]
			aci = (aci + 1) % len(availableColors)
			colors[pn] = color
		}
	}

	return colors
}

type output struct {
	sb     strings.Builder
	indent string
	prev   map[string]bool
}

func newOutput() *output {
	return &output{
		prev: map[string]bool{},
	}
}

func (o *output) addLine(format string, a ...any) {
	l := fmt.Sprintf(format, a...)

	if strings.HasPrefix(l, "}") {
		o.decreaseIndent()
	}

	o.sb.WriteString(o.indent + l + "\n")

	if strings.HasSuffix(l, "{") {
		o.increaseIndent()
	}
}

func (o *output) addLineDistinct(s any) {
	l := fmt.Sprint(s)

	if !o.prev[l] {
		o.addLine(l)
		o.prev[l] = true
	}
}

func (o *output) increaseIndent() {
	o.indent += "   "
}

func (o *output) decreaseIndent() {
	o.indent = o.indent[:len(o.indent)-3]
}

func (o *output) String() string {
	return o.sb.String()
}

type node struct {
	name    string
	files   int
	attribs map[string]string
}

func newNode(name string) *node {
	return &node{
		name:    name,
		attribs: map[string]string{},
	}
}

func (n *node) String() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf(`"%v"`, n.name))
	writeAttribs(&sb, n.attribs)
	sb.WriteString(";")

	return sb.String()
}

type edge struct {
	src     string
	dest    string
	attribs map[string]string
}

func newEdge(src, dest string) *edge {
	return &edge{
		src:     src,
		dest:    dest,
		attribs: map[string]string{},
	}
}

func (e *edge) String() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf(`"%v" -> "%v"`, e.src, e.dest))
	writeAttribs(&sb, e.attribs)
	sb.WriteString(";")

	return sb.String()
}

func writeAttribs(sb *strings.Builder, attribs map[string]string) {
	keys := sortedKeys(attribs)

	if len(keys) == 0 {
		return
	}

	sb.WriteString(" [")

	for _, k := range keys {
		v := attribs[k]

		if strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">") {
			sb.WriteString(fmt.Sprintf(` "%v"=%v`, k, v))
		} else {
			sb.WriteString(fmt.Sprintf(` "%v"="%v"`, k, v))
		}
	}

	sb.WriteString(" ]")
}

func sortedKeys(d map[string]string) []string {
	result := make([]string, 0, len(d))

	for k, v := range d {
		if v != "" {
			result = append(result, k)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}
