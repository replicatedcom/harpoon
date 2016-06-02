package log

import (
	"os"

	logging "github.com/op/go-logging"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	textFormatTTY = logging.MustStringFormatter(
		`%{color}%{level:.4s}%{color:reset} %{time:2006-01-02T15:04:05-07:00} %{shortfile} [%{module}] %{message}`,
	)
	textFormat = logging.MustStringFormatter(
		`%{level:.4s} %{time:2006-01-02T15:04:05-07:00} %{shortfile} [%{module}] %{message}`,
	)
)

func GetTextFormat() logging.Formatter {
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		return textFormatTTY
	} else {
		return textFormat
	}
}
