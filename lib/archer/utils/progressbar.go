package utils

import (
	"time"

	"github.com/schollz/progressbar/v3"
)

func NewProgressBar(total int) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionThrottle(time.Second),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{Saucer: "#", SaucerPadding: " ", BarStart: "|", BarEnd: "|"}),
		progressbar.OptionClearOnFinish(),
		// progressbar.OptionOnCompletion(func() {
		// 	_, _ = fmt.Fprint(os.Stdout, "\n")
		// }),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}
