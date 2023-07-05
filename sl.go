// The [sl] package has two main concepts, the [ServiceLocator] itself is the
// main object that one should pass around through the application. A
// [ServiceLocator] has a list of slots that can be filled with
// [ProvideFunc] and [Provide] and retrieved with [Use]. Slots should
// be unique by type and they can only be created with the [NewSlot] function.
//
// The usual way to use this module is to make slots for Go interfaces and
// then pass implementations using the [Provide] and
// [ProvideFunc] functions.
//
// Services can be of various types:
//   - a service with no dependencies can be directly injected inside a
//     ServiceLocator using [Provide].
//   - a service with dependencies on other service should use
//     [ProvideFunc]. This lets the service configure itself when needed
//     and makes the developer not think about correctly ordering the
//     recursive configuration of its dependencies.
//   - a service can also be private, in this case the slot for a service
//     should be a private field in the service package. This kind of
//     services should also provide a way to inject them into a
//     ServiceLocator.
//   - a package can also just provide a slot with some value. This is useful
//     for using the ServiceLocator to easily pass around values, effectively
//     threating slots just as dynamically scoped variables.
package sl

import (
	"fmt"
	"log"
	"os"
)

func zero[T any]() T {
	var zero T
	return zero
}

// Logger is the debug logger
//
// TODO: in the future this will be disabled and discard by default.
//
// As this is the service locator module it was meaning less to pass this
// through the ServiceLocator itself (without making the whole module more
// complex)
var Logger *log.Logger = log.New(os.Stderr, "[service locator] ", log.Lmsgprefix)

// slot is just a "typed" unique "symbol"
//
// This must be defined like so and not for example "struct{ typeName string }"
// because we might want to have more slots for the same type.
type slot[T any] *struct{}

// hook is just a "typed" unique "symbol"
//
// See [slot] for more information about this type
type hook[T any] *struct{}

type Hook[T any] func(*ServiceLocator, T) error

// NewSlot is the only way to create instances of the slot type. Each instance
// is unique.
//
// This then lets you attach a service instance of type "T" for this slot to a
// [ServiceLocator] object.
func NewSlot[T any]() slot[T] {
	return slot[T](new(struct{}))
}

// NewHook is the only way to create instances of the hook type. Each instance
// is unique.
//
// This lets you have a service dispatch an hook with a message of type "T".
func NewHook[T any]() hook[T] {
	return hook[T](new(struct{}))
}

// slotEntry represents a service that can lazily configured
// (using "configureFunc"). Once configured the instance is kept in the "value"
// field and "created" will always be "true". The field "typeName" just for
// debugging purposes.
type slotEntry struct {
	// typeName is just used for debugging purposes
	typeName string

	// configureFunc is used by lazily provided slot values to tell how to
	// configure them self when required
	configureFunc func(*ServiceLocator) (any, error)

	// configured tells if this slot is already configured
	configured bool

	// value for this slot
	value any
}

// ensureConfigured tries to call configure on this slot entry if not already configured
func (s *slotEntry) ensureConfigured(l *ServiceLocator) error {
	if !s.configured {
		v, err := s.configureFunc(l)
		if err != nil {
			return err
		}

		Logger.Printf(`[slot: %s] configured service of type %T`, s.typeName, v)

		s.configured = true
		s.value = v
	}

	return nil
}

type hookEntry struct {
	// typeName is just used for debugging purposes
	typeName string

	// listeners is a list of functions to call when this hook is called
	listeners []func(*ServiceLocator, any) error
}

// ServiceLocator is the main context passed around to retrive service
// instances.
//
// The interface uses generics so to inject and retrive service instances in a
// type-safe manner. You should use the functions [Provide], [ProvideFunc],
// [Use] and [Invoke] and their variations to interact with an instance of this
// type.
//
// This is essentially a dictionary of slots and hooks that are them self just
// uniquely typed symbols.
type ServiceLocator struct {
	providers map[any]*slotEntry
	hooks     map[any]*hookEntry
}

// New creates a new [ServiceLocator] context to pass around in the application.
func New() *ServiceLocator {
	return &ServiceLocator{
		providers: map[any]*slotEntry{},
		hooks:     map[any]*hookEntry{},
	}
}

//
// Slots
//

// Provide will inject a concrete instance inside the ServiceLocator "l" for
// the given "slotKey". This should be used for injecting "static" services, for
// instances whose construction depend on other services you should use the
// [ProvideFunc] function.
//
// This is generic over "T" to check that instances returned by the "createFunc"
// are compatible with "T" as it can also be an interface.
func Provide[T any](l *ServiceLocator, slotKey slot[T], value T) T {
	typeName := getTypeName[T]()

	Logger.Printf(`[slot: %s] provided value of type %T`, typeName, value)

	l.providers[slotKey] = &slotEntry{
		typeName:   typeName,
		configured: true,
		value:      value,
	}
	return value
}

// ProvideFunc will inject an instance inside the given ServiceLocator and
// "slotKey" that is created only when requested with a call to the [Use] or
// [Invoke] functions.
//
// This is generic over "T" to check that instances returned by the "createFunc"
// are compatible with "T" as it can also be an interface.
func ProvideFunc[T any](l *ServiceLocator, slotKey slot[T], createFunc func(*ServiceLocator) (T, error)) {
	typeName := getTypeName[T]()
	Logger.Printf(`[slot: %s] inject lazy provider`, typeName)

	l.providers[slotKey] = &slotEntry{
		typeName:      typeName,
		configureFunc: func(l *ServiceLocator) (any, error) { return createFunc(l) },
		configured:    false,
	}
}

// useSlotValue tries to configure the slot for slotKey and if done correctly returns it.
func useSlotValue[T any](l *ServiceLocator, slotKey slot[T]) (T, error) {
	slot, ok := l.providers[slotKey]
	if !ok {
		return zero[T](), fmt.Errorf(`no injected value for type %s`, getTypeName[T]())
	}

	if err := slot.ensureConfigured(l); err != nil {
		return zero[T](), err
	}

	return slot.value.(T), nil
}

// Use retrieves the value of type T associated with the given slot key from
// the provided [ServiceLocator] instance.
//
// If the [ServiceLocator] does not have a value for the slot key, or if the
// value wasn't correctly configured (in the case of a lazy slot), an error
// is returned.
func Use[T any](l *ServiceLocator, slotKey slot[T]) (T, error) {
	v, err := useSlotValue(l, slotKey)
	if err != nil {
		return zero[T](), err
	}

	return v, nil
}

// MustUse is the same as [Use] but panics if there is any error in locating the service
func MustUse[T any](l *ServiceLocator, slotKey slot[T]) T {
	v, err := useSlotValue(l, slotKey)
	if err != nil {
		panic(err)
	}

	return v
}

// Invoke is the same as [Use] but discards the value and just returns the error
func Invoke[T any](l *ServiceLocator, slotKey slot[T]) error {
	_, err := useSlotValue(l, slotKey)
	if err != nil {
		return err
	}

	return nil
}

// MustInvoke is the same as [Invoke] but panics if there is any error in locating the service
func MustInvoke[T any](l *ServiceLocator, slotKey slot[T]) {
	if _, err := useSlotValue(l, slotKey); err != nil {
		panic(err)
	}
}

//
// Hooks
//

// ProvideHook attaches a list of ordered listeners to a given hook of type "T".
// This is supposed to be called when composing the full application on an high
// level.
//
// For example to easily enable or disable routes in an http server based on
// some environment variables when setting up the application.
func ProvideHook[T any](l *ServiceLocator, hookKey hook[T], listeners ...Hook[T]) {
	typeName := getTypeName[T]()
	Logger.Printf(`[hook: %s] injecting hooks`, typeName)

	// cast type safe listeners to internal untyped version to put inside the hook map
	anyListeners := make([]func(*ServiceLocator, any) error, len(listeners))
	for i, l := range listeners {
		ll := l
		anyListeners[i] = func(l *ServiceLocator, a any) error {
			t, ok := a.(T)
			if !ok {
				panic(`illegal state`)
			}

			return ll(l, t)
		}
	}

	l.hooks[hookKey] = &hookEntry{
		typeName:  typeName,
		listeners: anyListeners,
	}
}

// UseHook is supposed to be used by services to dispatch some action during the
// creation of the application.
//
// For example to attach some routes to a given router in a deterministic order
// a composable manner.
func UseHook[T any](l *ServiceLocator, hookKey hook[T], value T) error {
	hookEntry, ok := l.hooks[hookKey]
	if !ok {
		return fmt.Errorf(`no injected hooks for hook of type %s`, hookEntry.typeName)
	}

	Logger.Printf(`[hook: %s] calling hook with value of type %T`, hookEntry.typeName, value)
	for _, hookFunc := range hookEntry.listeners {
		if err := hookFunc(l, value); err != nil {
			return err
		}
	}

	return nil
}

// MustUseHook is the same as [UseHook] but panics if there is some error
func MustUseHook[T any](l *ServiceLocator, hookKey hook[T], value T) {
	if err := UseHook(l, hookKey, value); err != nil {
		panic(err)
	}
}

// getTypeName is a trick to get the name of a type (even if it is an
// interface type)
func getTypeName[T any]() string {
	var zero T
	return fmt.Sprintf(`%T`, &zero)[1:]
}
