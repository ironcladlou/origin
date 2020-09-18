package namespace

type Done func()
type Getter func() (namespace string, done Done, err error)

type Pool interface {
	GetNamespace() (namespace string, done Done, err error)
	Dispose() error
}
