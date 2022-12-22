package proxy

type LocalStorage interface {
	Get(key string) (string, error)
	Insert(url string) string
}

type ProxyService struct {
	storage LocalStorage
}

func New(storage LocalStorage) ProxyService {
	return ProxyService{
		storage: storage,
	}
}

func (p *ProxyService) CreateRedirect(key string) string {
	return p.storage.Insert(key)
}

func (p *ProxyService) GetLinkByKeyID(key string) (string, error) {
	return p.storage.Get(key)
}
