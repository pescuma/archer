package importers

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
	"github.com/Faire/archer/lib/archer/utils"
)

type work struct {
	index    int
	path     string
	contents string
	result   interface{}
}

func ProcessKotlinFiles[T any](paths []string,
	process func(file string, content kotlin_parser.IKotlinFileContext) (T, error),
	onError func(file string, err error) error,
) ([]T, error) {
	bar := utils.NewProgressBar(len(paths))

	group := utils.NewProcessGroup(func(w *work) (*work, error) {
		el := antlrErrorListener{}
		input := antlr.NewInputStream(w.contents)

		lexer := kotlin_parser.NewKotlinLexer(input)
		lexer.RemoveErrorListeners()
		lexer.AddErrorListener(&el)

		stream := antlr.NewCommonTokenStream(lexer, 0)

		parser := kotlin_parser.NewKotlinParser(stream)
		parser.RemoveErrorListeners()
		parser.AddErrorListener(&el)

		content := parser.KotlinFile()

		if el.errors != nil {
			err := errors.New(strings.Join(el.errors, ", "))

			_ = bar.Clear()
			err = onError(w.path, err)

			return w, err
		}

		result, err := process(w.path, content)
		if err != nil {
			_ = bar.Clear()
			err = onError(w.path, err)

			return w, err
		}

		w.result = result

		return w, err
	})

	go func() {
		for i, path := range paths {
			contents, err := os.ReadFile(path)
			if err != nil {
				_ = bar.Clear()
				err = onError(path, err)
				if err != nil {
					group.Abort(err)
					return
				}

				group.Output <- &work{
					index: i,
					path:  path,
				}

			} else {
				group.Input <- &work{
					index:    i,
					path:     path,
					contents: string(contents),
				}
			}
		}

		group.FinishedInput()
	}()

	result := make([]T, len(paths))
	for w := range group.Output {
		_ = bar.Add(1)

		if w.result != nil {
			result[w.index] = w.result.(T)
		}
	}

	if err := <-group.Err; err != nil {
		return nil, err
	}

	return result, nil
}

type antlrErrorListener struct {
	antlr.DefaultErrorListener
	errors []string
}

func (d *antlrErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	d.errors = append(d.errors, fmt.Sprintf("line "+strconv.Itoa(line)+":"+strconv.Itoa(column)+" "+msg))
}
