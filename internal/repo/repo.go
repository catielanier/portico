package repo

type RepositoryKind string

const (
	RepositoryKindOfficial RepositoryKind = "official"
	RepositoryKindOverlay  RepositoryKind = "overlay"
	RepositoryKindLocal    RepositoryKind = "local"
	RepositoryKindUnknown  RepositoryKind = "unknown"
)

type Repository struct {
	Name        string
	Kind        RepositoryKind
	Enabled     bool
	Description string
	Location    string
	SyncURI     string
	Priority    *int
}

type AddRequest struct {
	Name    string
	SyncURI string
	Kind    RepositoryKind
}

type RemoveRequest struct {
	Name string
}

type Manager interface {
	List() ([]Repository, error)
	Add(req AddRequest) error
	Remove(req RemoveRequest) error
}

type StubManager struct{}

func NewStubManager() *StubManager {
	return &StubManager{}
}

func (m *StubManager) List() ([]Repository, error) {
	return []Repository{
		{
			Name:        "gentoo",
			Kind:        RepositoryKindOfficial,
			Enabled:     true,
			Description: "Official Gentoo repository",
		},
		{
			Name:        "guru",
			Kind:        RepositoryKindOverlay,
			Enabled:     false,
			Description: "Gentoo User Repository",
		},
	}, nil
}

func (m *StubManager) Add(req AddRequest) error {
	return nil
}

func (m *StubManager) Remove(req RemoveRequest) error {
	return nil
}
