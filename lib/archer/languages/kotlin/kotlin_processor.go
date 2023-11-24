package kotlin

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"

	"github.com/pescuma/archer/lib/archer/languages/kotlin_parser"
	"github.com/pescuma/archer/lib/archer/utils"
)

type work struct {
	index    int
	path     string
	contents []byte
	err      error
}

func ProcessFiles(paths []string,
	process func(file string, content kotlin_parser.IKotlinFileContext) error,
	onProcessed func(bar *progressbar.ProgressBar, index int, file string) error,
	onError func(bar *progressbar.ProgressBar, index int, file string, err error) error,
) error {
	group := utils.NewProcessGroup(
		func(w *work) (*work, error) {
			contents := string(w.contents)
			w.contents = nil

			el := antlrErrorListener{}
			input := antlr.NewInputStream(contents)

			lexer := kotlin_parser.NewKotlinLexer(input)
			lexer.RemoveErrorListeners()
			lexer.AddErrorListener(&el)

			stream := antlr.NewCommonTokenStream(lexer, 0)

			parser := kotlin_parser.NewKotlinParser(stream)
			parser.RemoveErrorListeners()
			parser.AddErrorListener(&el)

			content := parser.KotlinFile()

			if el.errors != nil {
				w.err = errors.New(strings.Join(el.errors, ", "))
				return w, nil
			}

			err := process(w.path, content)
			if err != nil {
				w.err = err
				return w, nil
			}

			return w, nil
		})

	go func() {
		for i, path := range paths {
			contents, err := os.ReadFile(path)
			if err != nil {
				group.Output <- &work{
					index: i,
					path:  path,
					err:   err,
				}
				continue
			}

			group.Input <- &work{
				index:    i,
				path:     path,
				contents: contents,
			}
		}

		group.FinishedInput()
	}()

	bar := utils.NewProgressBar(len(paths))
	index := 0
	for w := range group.Output {
		if w.err == nil {
			err := onProcessed(bar, index, w.path)
			if err != nil {
				group.Abort(err)
			}

		} else {
			err := onError(bar, index, w.path, w.err)
			if err != nil {
				group.Abort(err)
			}
		}

		index++
		_ = bar.Add(1)
	}

	if err := <-group.Err; err != nil {
		return err
	}

	return nil
}

type antlrErrorListener struct {
	*antlr.DefaultErrorListener
	errors []string
}

func (d *antlrErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	d.errors = append(d.errors, fmt.Sprintf("line "+strconv.Itoa(line)+":"+strconv.Itoa(column)+" "+msg))
}
