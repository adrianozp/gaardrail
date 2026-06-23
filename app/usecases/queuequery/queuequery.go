package queuequery

//go:generate mockery --all --output=mocks --outpkg=mocks
type QueryHolder interface {
	SetQuery(string)
	Query() string
}

type UseCase struct {
	holder QueryHolder
}

func New(h QueryHolder) UseCase {
	return UseCase{holder: h}
}

func (u UseCase) Set(query string) {
	u.holder.SetQuery(query)
}

func (u UseCase) Get() string {
	return u.holder.Query()
}
