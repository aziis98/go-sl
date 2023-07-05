package sl_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aziis98/go-sl"
	"gotest.tools/assert"
)

type Config struct {
	Foo string
}

var ConfigSlot = sl.NewSlot[*Config]()

type ExampleService struct {
	Bar    string
	Logger *log.Logger
}

var ExampleServiceSlot = sl.NewSlot[*ExampleService]()

var LoggerSlot = sl.NewSlot[*log.Logger]()

func TestBasic(t *testing.T) {
	l := sl.New()

	sl.ProvideFunc(l, ConfigSlot, func(sl *sl.ServiceLocator) (*Config, error) {
		// foo, ok := os.LookupEnv("FOO")
		// if !ok {
		// 	foo = ""
		// }
		return &Config{
			// Foo: foo,
			Foo: "foo",
		}, nil
	})

	sl.ProvideFunc(l, LoggerSlot, func(l *sl.ServiceLocator) (*log.Logger, error) {
		config := sl.MustUse(l, ConfigSlot)

		return log.New(os.Stderr, fmt.Sprintf("[service foo=%s] ", config.Foo), log.Lmsgprefix), nil
	})

	sl.ProvideFunc(l, ExampleServiceSlot, func(l *sl.ServiceLocator) (*ExampleService, error) {
		config, err := sl.Use(l, ConfigSlot)
		if err != nil {
			return nil, err
		}
		logger, err := sl.Use(l, LoggerSlot)
		if err != nil {
			return nil, err
		}

		logger.Printf(`creating the service`)

		return &ExampleService{
			Bar: config.Foo + " baz",
		}, nil
	})

	example, err := sl.Use(l, ExampleServiceSlot)
	if err != nil {
		t.Fatal(err)
	}

	assert.DeepEqual(t, example, &ExampleService{
		Bar: "foo baz",
	})
}
