package params

import (
	goflag "flag"
	"os"
	"reflect"
	"strings"

	"github.com/namsral/flag"
)

type flagValueWrapper struct {
	inner    goflag.Value
	flagType string
}

func wrapFlagValue(v goflag.Value) flag.Value {
	if pv, ok := v.(flag.Value); ok {
		return pv
	}

	pv := &flagValueWrapper{
		inner: v,
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Interface || t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	pv.flagType = strings.TrimSuffix(t.Name(), "Value")
	return pv
}

func (v *flagValueWrapper) String() string {
	return v.inner.String()
}

func (v *flagValueWrapper) Set(s string) error {
	return v.inner.Set(s)
}

func (v *flagValueWrapper) Type() string {
	return v.flagType
}

func AddGoFlag(newSet *flag.FlagSet, goflag *goflag.Flag) {
	newSet.Var(wrapFlagValue(goflag.Value), goflag.Name, goflag.Usage)
}

func FlagSetFromGoFlagSet(flagSet *goflag.FlagSet) *flag.FlagSet {
	newSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.VisitAll(func(goflag *goflag.Flag) {
		AddGoFlag(newSet, goflag)
	})
	return newSet
}
