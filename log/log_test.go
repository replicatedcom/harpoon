package log

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	logging "github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func (s *LogTestSuite) TestLogCallDepth() {
	t := s.T()
	Debugf("test file")
	parts := strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "log_test.go:17", parts[0])
}

func (s *LogTestSuite) TestLogf() {
	t := s.T()

	Debugf("test %s level log", "debug")
	parts := strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "DEBUG", parts[1])
	assert.Equal(t, "test debug level log\n", parts[2])
	s.Buf.Reset()

	Infof("test %s level log", "info")
	parts = strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "INFO", parts[1])
	assert.Equal(t, "test info level log\n", parts[2])
	s.Buf.Reset()

	Warningf("test %s level log", "warning")
	parts = strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "WARNING", parts[1])
	assert.Equal(t, "test warning level log\n", parts[2])
	s.Buf.Reset()

	Errorf("test %s level log", "error")
	parts = strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "ERROR", parts[1])
	assert.Equal(t, "test error level log\n", parts[2])
	s.Buf.Reset()
}

func (s *LogTestSuite) TestLogErr() {
	t := s.T()

	err := errors.New("error to log")

	Warning(err)
	parts := strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "WARNING", parts[1])
	assert.Equal(t, "error to log\n", parts[2])
	s.Buf.Reset()

	Error(err)
	parts = strings.SplitN(s.Buf.String(), " ", 3)
	assert.Equal(t, "ERROR", parts[1])
	assert.Equal(t, "error to log\n", parts[2])
	s.Buf.Reset()
}

type LogTestSuite struct {
	suite.Suite
	Buf *bytes.Buffer
}

func (s *LogTestSuite) SetupTest() {
	s.Buf = &bytes.Buffer{}
	var backend logging.Backend = logging.NewLogBackend(s.Buf, "", 0)
	backend = logging.NewBackendFormatter(backend, logging.MustStringFormatter("%{shortfile} %{level} %{message}"))
	log.SetBackend(logging.AddModuleLevel(backend))
}

func (s *LogTestSuite) TearDownSuite() {
	log.SetBackend(logging.AddModuleLevel(logging.NewLogBackend(os.Stderr, "", 0)))
}

func TestLogTestSuite(t *testing.T) {
	suite.Run(t, new(LogTestSuite))
}
