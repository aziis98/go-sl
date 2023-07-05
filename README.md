# Go Service Locator

A small [service locator](https://en.wikipedia.org/wiki/Service_locator_pattern) library with the following features

- Uses generics to provide a type safe interface without using reflection.

- Automatically resolves dependency initialization like [dig](https://github.com/uber-go/dig).

- An "hooks" system to easily specify the order of setup operations in a deterministic order.

## Usage

Install this package with

```bash shell
$ go get github.com/aziis98/go-sl
```

### Simple Example

```go
// Define each service struct or interface with a corresponding slot
var ConfigSlot   = sl.NewSlot[*Config]()
var LoggerSlot   = sl.NewSlot[*log.Logger]()
var DatabaseSlot = sl.NewSlot[Database]()
var HandlerSlot  = sl.NewSlot[Handler]()
var ServerSlot   = sl.NewSlot[*Server]()
...

func main() {
    l := sl.New()

    sl.ProvideFunc(l, ConfigSlot, func(sl *sl.ServiceLocator) (*Config, error) {
        err := godotenv.Load()
        ...
        
        return &Config{ ... }, nil
    })
    
    sl.ProvideFunc(l, LoggerSlot, func(l *sl.ServiceLocator) (*log.Logger, error) {
        config := sl.MustUse(l, ConfigSlot)
        return log.New(os.Stderr, fmt.Sprintf("[service foo=%s] ", config.Foo), log.Lmsgprefix), nil
    })

    sl.ProvideFunc(l, ServerSlot, func(l *sl.ServiceLocator) (*Server, error) {
        config, err := sl.Use(l, ConfigSlot)
        if err != nil {
            return nil, err
        }
        handler, err := sl.Use(l, HandlerSlot)
        if err != nil {
            return nil, err
        }

        return &Server{ ... }, nil
    })

    srv := sl.MustUse(l, ServerSlot)
    log.Fatal(srv.ListenAndServe())
}
```

### Medium Example

For example a `database` module / service can be defined as follows

```go
package database

type Database interface {
    CreatePost(content string) (uint, error)
    ReadPost(id uint) (string, error)
    ReadAllPosts() ([]string, error)
    UpdatePost(id uint, newContent string) error
    DeletePost(id uint) error
}

var Slot = sl.NewSlot[Database]()
```

We can then have various implementations like `database/mock.go`

```go
package database

type Mock struct {
    posts map[uint]string
}

func (db *Mock) CreatePost(content string) (uint, error) { ... }
func (db *Mock) ReadPost(id uint) (string, error) { ... }
func (db *Mock) ReadAllPosts() ([]string, error) { ... }
func (db *Mock) UpdatePost(id uint, newContent string) error { ... }
func (db *Mock) DeletePost(id uint) error { ... }

func NewMockDatabase() *Mock {
    return &Mock{ ... }
}

func ConfigureMockDatabase(l *sl.ServiceLocator) (Database, error) {
    config, err := sl.Use(l, config.Slot)
    if err != nil {
        return nil, err
    }

    mock := NewMockDatabase()
    ...
    return mock, nil
}
```

And then in the main call

```go
func main() {
    ...
    sl.ProvideFunc(l, database.Slot, database.ConfigureMockDatabase)
    ...
}
```

## Theory

[...]

### Introduction

[...]

### Advanced Slots

[...]

### Hooks

[...]

