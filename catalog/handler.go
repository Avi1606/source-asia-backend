package catalog

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type Handler struct {
	store *CatalogStore
}

func NewHandler(store *CatalogStore) *Handler {
	return &Handler{store: store}
}

type productRequest struct {
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	ImageURLs *[]string `json:"image_urls"`
	VideoURLs *[]string `json:"video_urls"`
}

type mediaRequest struct {
	ImageURLs *[]string `json:"image_urls"`
	VideoURLs *[]string `json:"video_urls"`
}

type productDetailResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	SKU        string   `json:"sku"`
	ImageCount int      `json:"image_count"`
	VideoCount int      `json:"video_count"`
	CreatedAt  string   `json:"created_at"`
	ImageURLs  []string `json:"image_urls"`
	VideoURLs  []string `json:"video_urls"`
}

type listProductsResponse struct {
	Data       []Product          `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

type paginationResponse struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var body productRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	input, err := ValidateProductInput(body.Name, body.SKU, body.ImageURLs, body.VideoURLs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, media, err := h.store.CreateProduct(input.Name, input.SKU, input.ImageURLs, input.VideoURLs)
	if errors.Is(err, ErrDuplicateSKU) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "sku already exists"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create product"})
		return
	}

	writeJSON(w, http.StatusCreated, detailResponse(product, media))
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	products, total := h.store.ListProducts(limit, offset)
	writeJSON(w, http.StatusOK, listProductsResponse{
		Data: products,
		Pagination: paginationResponse{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	product, media, ok := h.store.GetProduct(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	writeJSON(w, http.StatusOK, detailResponse(product, media))
}

func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	var body mediaRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	images, videos, err := ValidateMediaInput(body.ImageURLs, body.VideoURLs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, media, err := h.store.AddMedia(r.PathValue("id"), images, videos)
	if errors.Is(err, ErrProductNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add media"})
		return
	}

	writeJSON(w, http.StatusOK, detailResponse(product, media))
}

func parsePagination(r *http.Request) (int, int, error) {
	query := r.URL.Query()
	limit := 20
	offset := 0

	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, errors.New("limit must be a positive integer")
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}

	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
		offset = parsed
	}

	return limit, offset, nil
}

func detailResponse(product Product, media ProductMedia) productDetailResponse {
	return productDetailResponse{
		ID:         product.ID,
		Name:       product.Name,
		SKU:        product.SKU,
		ImageCount: product.ImageCount,
		VideoCount: product.VideoCount,
		CreatedAt:  product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		ImageURLs:  media.ImageURLs,
		VideoURLs:  media.VideoURLs,
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}

	var extra struct{}
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("request body must contain a single JSON object")
	} else if !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON body")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
