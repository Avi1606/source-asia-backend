package catalog

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var ErrProductNotFound = errors.New("product not found")
var ErrDuplicateSKU = errors.New("sku already exists")

type Product struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	SKU        string    `json:"sku"`
	ImageCount int       `json:"image_count"`
	VideoCount int       `json:"video_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type ProductMedia struct {
	ImageURLs []string
	VideoURLs []string
}

type CatalogStore struct {
	mu       sync.RWMutex
	products map[string]*Product
	media    map[string]*ProductMedia
	skuIndex map[string]string
	ordered  []string
}

func NewCatalogStore() *CatalogStore {
	return &CatalogStore{
		products: make(map[string]*Product),
		media:    make(map[string]*ProductMedia),
		skuIndex: make(map[string]string),
		ordered:  make([]string, 0),
	}
}

func (s *CatalogStore) CreateProduct(name, sku string, imageURLs, videoURLs []string) (Product, ProductMedia, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.skuIndex[sku]; exists {
		return Product{}, ProductMedia{}, ErrDuplicateSKU
	}

	id, err := newUUIDV4()
	if err != nil {
		return Product{}, ProductMedia{}, err
	}

	product := &Product{
		ID:         id,
		Name:       name,
		SKU:        sku,
		ImageCount: len(imageURLs),
		VideoCount: len(videoURLs),
		CreatedAt:  time.Now().UTC(),
	}
	media := &ProductMedia{
		ImageURLs: cloneStrings(imageURLs),
		VideoURLs: cloneStrings(videoURLs),
	}

	s.products[id] = product
	s.media[id] = media
	s.skuIndex[sku] = id
	s.ordered = append(s.ordered, id)

	return *product, copyMedia(media), nil
}

func (s *CatalogStore) ListProducts(limit, offset int) ([]Product, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.ordered)
	if offset > total {
		return []Product{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	products := make([]Product, 0, end-offset)
	for _, id := range s.ordered[offset:end] {
		products = append(products, *s.products[id])
	}

	return products, total
}

func (s *CatalogStore) GetProduct(id string) (Product, ProductMedia, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, ok := s.products[id]
	if !ok {
		return Product{}, ProductMedia{}, false
	}

	return *product, copyMedia(s.media[id]), true
}

func (s *CatalogStore) AddMedia(id string, imageURLs, videoURLs []string) (Product, ProductMedia, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, ok := s.products[id]
	if !ok {
		return Product{}, ProductMedia{}, ErrProductNotFound
	}

	media := s.media[id]
	media.ImageURLs = append(media.ImageURLs, imageURLs...)
	media.VideoURLs = append(media.VideoURLs, videoURLs...)
	product.ImageCount = len(media.ImageURLs)
	product.VideoCount = len(media.VideoURLs)

	return *product, copyMedia(media), nil
}

func copyMedia(media *ProductMedia) ProductMedia {
	if media == nil {
		return ProductMedia{
			ImageURLs: []string{},
			VideoURLs: []string{},
		}
	}
	return ProductMedia{
		ImageURLs: cloneStrings(media.ImageURLs),
		VideoURLs: cloneStrings(media.VideoURLs),
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func newUUIDV4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(b[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
