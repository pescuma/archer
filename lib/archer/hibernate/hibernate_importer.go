package hibernate

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/common"
	"github.com/Faire/archer/lib/archer/kotlin_parser"
	"github.com/Faire/archer/lib/archer/model"
	"github.com/Faire/archer/lib/archer/utils"
)

type hibernateImporter struct {
	storage     archer.Storage
	rootsFinder common.RootsFinder
	rootName    string
}

func NewImporter(rootDirs, globs []string, rootName string) archer.Importer {
	return &hibernateImporter{
		rootsFinder: common.NewRootsFinder(rootDirs, globs),
		rootName:    rootName,
	}
}

func (i *hibernateImporter) Import(projs *model.Projects, files *model.Files, storage archer.Storage) error {
	i.storage = storage

	roots, err := i.rootsFinder.ComputeRootDirs(projs, files)
	if err != nil {
		return err
	}

	for _, r := range roots {
		fmt.Printf("%v\n", r)
	}

	type work struct {
		root         common.RootDir
		fileName     string
		fileContents string
		classes      map[string]*classInfo
		errors       []string
	}

	group := utils.NewProcessGroup(func(w *work) (*work, error) {
		var err error
		w.classes, w.errors, err = i.processKotlin(w.fileContents, w.fileName, w.root)
		return w, err
	})

	go func() {
		for _, root := range roots {
			err = root.WalkDir(func(proj *model.Project, path string) error {
				if group.Aborted() {
					return errors.New("aborted")
				}

				if !strings.HasSuffix(path, ".kt") {
					return nil
				}

				contents, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				group.Input <- &work{
					root:         root,
					fileName:     path,
					fileContents: string(contents),
				}

				return nil
			})
			if err != nil {
				group.Abort(err)
				return
			}
		}

		group.FinishedInput()
	}()

	classes := map[string]*classInfo{}
	var es []string
	for w := range group.Output {
		for k, nv := range w.classes {
			ov, ok := classes[k]
			if !ok {
				classes[k] = nv

			} else {
				ov.Root = append(ov.Root, nv.Root...)
				ov.Paths = append(ov.Paths, nv.Paths...)
				ov.Tables = append(ov.Tables, nv.Tables...)
				ov.Dependencies = append(ov.Dependencies, nv.Dependencies...)
			}
		}

		es = append(es, w.errors...)
	}

	if err = <-group.Err; err != nil {
		return err
	}

	for _, e := range es {
		fmt.Printf("ERROR: %v\n", e)
	}

	dbProjs := map[*model.Project]bool{}

	for _, c := range classes {
		if len(c.Paths) != 1 {
			fmt.Printf("ERROR: %v is associated with more than one path: %v. Ignoring!\n", c.Name, strings.Join(c.Paths, ", "))
			continue
		}
		if len(c.Tables) != 1 {
			fmt.Printf("ERROR: %v is associated with more than one table: %v. Ignoring!\n", c.Name, strings.Join(c.Tables, ", "))
			continue
		}

		root := c.Root[0]

		proj := projs.Get(i.rootName, c.Tables[0])
		dbProjs[proj] = true

		proj.Type = model.DatabaseType
		proj.ProjectFile = c.Paths[0]

		if root.Dir != nil {
			proj.RootDir = *root.Dir
		} else {
			proj.RootDir = root.Project.RootDir

			parent := root.Project

			parent.GetDependency(proj)
		}

		for _, di := range c.Dependencies {
			dc, ok := classes[di.ClassName]
			if !ok {
				fmt.Printf("ERROR: %v depends on unknown class: %v. Ignoring!\n", c.Name, di.ClassName)
				continue
			}

			if len(dc.Paths) != 1 {
				fmt.Printf("ERROR: %v depends on %v wich is associated with more than one path: %v. Ignoring!\n", c.Name, di.ClassName, strings.Join(dc.Paths, ", "))
				continue
			}
			if len(dc.Tables) != 1 {
				fmt.Printf("ERROR: %v depends on %v wich is associated with more than one table: %v. Ignoring!\n", c.Name, di.ClassName, strings.Join(dc.Tables, ", "))
				continue
			}

			dp := projs.Get(i.rootName, dc.Tables[0])
			dbProjs[dp] = true

			d := proj.GetDependency(dp)

			if di.Lazy {
				d.SetData("type", "lazy")
				d.SetData("style", "dashed")
			}
		}
	}

	common.CreateTableNameParts(lo.Keys(dbProjs))

	return storage.WriteProjects(projs, archer.ChangedBasicInfo|archer.ChangedDependencies)
}

func (i *hibernateImporter) processKotlin(fileContents string, fileName string, root common.RootDir) (map[string]*classInfo, []string, error) {
	l := newTreeListener(fileName, root)

	l.printfln("Parsing %v ...", fileName)
	l.IncreasePrefix()
	defer func() {
		l.DecreasePrefix()
		fmt.Print(l.sb.String())
	}()

	input := antlr.NewInputStream(fileContents)
	lexer := kotlin_parser.NewKotlinLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)

	p := kotlin_parser.NewKotlinParser(stream)

	file := p.KotlinFile()

	antlr.ParseTreeWalkerDefault.Walk(l, file)

	return l.Classes, l.Errors, nil
}

type treeListener struct {
	*kotlin_parser.BaseKotlinParserListener
	root             common.RootDir
	currentPath      string
	currentClassName []string
	currentClass     []*classInfo
	insideFunction   int
	insideProperty   bool

	hasColumnAnnotation bool
	hasLazyAnnotation   bool
	currentVariableType string

	sb     strings.Builder
	prefix string

	Classes map[string]*classInfo
	Errors  []string
}

type classInfo struct {
	Name         string
	Root         []common.RootDir
	Paths        []string
	Tables       []string
	Dependencies []*dependencyInfo
}

type dependencyInfo struct {
	ClassName string
	Lazy      bool
}

var tableRE = regexp.MustCompile(`name\s*=\s*"([^'"]+)"`)
var genericContainerRE = regexp.MustCompile(`Of<([^>]+)>\(`)

func newTreeListener(path string, root common.RootDir) *treeListener {
	return &treeListener{
		currentPath: path,
		root:        root,
		Classes:     map[string]*classInfo{},
	}
}

func (l *treeListener) printfln(format string, a ...any) {
	l.sb.WriteString(fmt.Sprintf(l.prefix+format+"\n", a...))
}

func (l *treeListener) IncreasePrefix() {
	l.prefix += "   "
}

func (l *treeListener) DecreasePrefix() {
	l.prefix = l.prefix[:len(l.prefix)-3]
}

func (l *treeListener) EnterClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	name := ctx.SimpleIdentifier().GetText()
	if len(l.currentClassName) > 0 {
		name = utils.Last(l.currentClassName) + "." + name
	}

	l.currentClassName = append(l.currentClassName, name)
	l.currentClass = append(l.currentClass, nil)

	l.printfln("found class %v", name)
	l.IncreasePrefix()
}

func (l *treeListener) ExitClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	l.DecreasePrefix()

	l.currentClassName = utils.RemoveLast(l.currentClassName)
	l.currentClass = utils.RemoveLast(l.currentClass)
}

func (l *treeListener) EnterFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	l.insideFunction++
}

func (l *treeListener) ExitFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	l.insideFunction--
}

func (l *treeListener) EnterPropertyDeclaration(ctx *kotlin_parser.PropertyDeclarationContext) {
	if l.insideFunction > 0 || len(l.currentClass) == 0 || utils.Last(l.currentClass) == nil {
		return
	}

	l.insideProperty = true
	l.hasColumnAnnotation = false
	l.hasLazyAnnotation = false
	l.currentVariableType = ""

	if ctx.VariableDeclaration() == nil {
		panic(fmt.Sprintf("Only supported one variable per property declaration (in %v %v)",
			l.currentPath, utils.Last(l.currentClassName)))
	}
}

func (l *treeListener) ExitPropertyDeclaration(ctx *kotlin_parser.PropertyDeclarationContext) {
	if !l.insideProperty {
		return
	}
	l.insideProperty = false

	if !l.hasColumnAnnotation {
		return
	}

	if l.currentVariableType == "" {
		if ctx.Expression() != nil {
			exp := ctx.Expression().GetText()
			ms := genericContainerRE.FindStringSubmatch(exp)
			if ms != nil {
				l.currentVariableType = ms[1]
			}
		}
	}

	l.currentVariableType = cleanTypeName(l.currentVariableType)

	varDecl := ctx.VariableDeclaration()

	l.printfln("found field %v", varDecl.GetText())
	l.IncreasePrefix()

	if l.currentVariableType == "" {
		l.printfln("could not find type of field")
		l.Errors = append(l.Errors, fmt.Sprintf("Could not find type of field %v %v %v",
			l.currentPath, utils.Last(l.currentClassName), varDecl.GetText()))

	} else {
		l.addDependency(l.currentVariableType, l.hasLazyAnnotation)
	}

	l.DecreasePrefix()
}

func (l *treeListener) ExitVariableDeclaration(ctx *kotlin_parser.VariableDeclarationContext) {
	if type_ := ctx.Type_(); type_ != nil {
		l.currentVariableType = type_.GetText()
	}
}

func (l *treeListener) EnterUnescapedAnnotation(ctx *kotlin_parser.UnescapedAnnotationContext) {
	text := ctx.GetText()
	parts := strings.SplitN(text, "(", 2)
	if len(parts) == 1 {
		parts = append(parts, "")
	}

	if parts[0] == "Table" {
		ms := tableRE.FindStringSubmatch(parts[1])
		if ms != nil {
			l.addTable(ms[1])
		}

	} else if parts[0] == "JoinColumn" {
		l.hasColumnAnnotation = true

	} else if parts[0] == "ManyToOne" || parts[0] == "OneToMany" || parts[0] == "OneToOne" {
		l.hasLazyAnnotation = strings.Index(parts[1], "FetchType.LAZY") >= 0
	}
}

func (l *treeListener) addTable(tableName string) {
	l.printfln("adding table: %v", tableName)

	cls := l.getClass(utils.Last(l.currentClassName))
	cls.Root = append(cls.Root, l.root)
	cls.Paths = append(cls.Paths, l.currentPath)
	cls.Tables = append(cls.Tables, tableName)

	l.currentClass[len(l.currentClass)-1] = cls
}

func (l *treeListener) addDependency(dependencyTypeName string, lazy bool) {
	l.printfln("adding dep: %v%v", dependencyTypeName, utils.IIf(lazy, " (lazy)", ""))

	cls := utils.Last(l.currentClass)
	cls.Dependencies = append(cls.Dependencies, &dependencyInfo{
		ClassName: dependencyTypeName,
		Lazy:      lazy,
	})
}

func (l *treeListener) getClass(name string) *classInfo {
	result, ok := l.Classes[name]

	if !ok {
		result = &classInfo{
			Name: name,
		}
		l.Classes[name] = result
	}

	return result
}

var genericRE = regexp.MustCompile(`^([^<]+)<(.*?)>\??$`)

func cleanTypeName(t string) string {
	t = strings.TrimSpace(t)

	for {
		matches := genericRE.FindStringSubmatch(t)
		if matches == nil {
			break
		}

		t1 := matches[1]
		t2 := matches[2]

		if t1 == "MutableList" || t1 == "List" || t1 == "MutableSet" || t1 == "Set" {
			t = t2

		} else {
			t = t1
			break
		}
	}

	t = strings.TrimSuffix(t, "?")
	return t
}
