package catalog

import (
	"fmt"
	"strings"
)

const (
	MaxURLsPerRequest = 20
	MaxURLLength      = 2048
)

type ProductInput struct {
	Name      string
	SKU       string
	ImageURLs []string
	VideoURLs []string
}

func ValidateProductInput(name, sku string, imageURLs, videoURLs *[]string) (ProductInput, error) {
	name = strings.TrimSpace(name)
	sku = strings.TrimSpace(sku)

	if name == "" {
		return ProductInput{}, fmt.Errorf("name is required and must be non-empty")
	}
	if sku == "" {
		return ProductInput{}, fmt.Errorf("sku is required and must be non-empty")
	}

	images, err := validateURLField("image_urls", imageURLs)
	if err != nil {
		return ProductInput{}, err
	}
	videos, err := validateURLField("video_urls", videoURLs)
	if err != nil {
		return ProductInput{}, err
	}

	return ProductInput{
		Name:      name,
		SKU:       sku,
		ImageURLs: images,
		VideoURLs: videos,
	}, nil
}

func ValidateMediaInput(imageURLs, videoURLs *[]string) ([]string, []string, error) {
	images, err := validateURLField("image_urls", imageURLs)
	if err != nil {
		return nil, nil, err
	}
	videos, err := validateURLField("video_urls", videoURLs)
	if err != nil {
		return nil, nil, err
	}
	if len(images) == 0 && len(videos) == 0 {
		return nil, nil, fmt.Errorf("image_urls or video_urls must contain at least one URL")
	}
	return images, videos, nil
}

func validateURLField(field string, urls *[]string) ([]string, error) {
	if urls == nil {
		return []string{}, nil
	}

	if len(*urls) > MaxURLsPerRequest {
		return nil, fmt.Errorf("%s must contain no more than %d URLs", field, MaxURLsPerRequest)
	}

	result := make([]string, len(*urls))
	for i, url := range *urls {
		if strings.TrimSpace(url) == "" {
			return nil, fmt.Errorf("%s[%d] must be a non-empty string", field, i)
		}
		if len(url) > MaxURLLength {
			return nil, fmt.Errorf("%s[%d] must be no longer than %d characters", field, i, MaxURLLength)
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return nil, fmt.Errorf("%s[%d] must start with http:// or https://", field, i)
		}
		result[i] = url
	}

	return result, nil
}
