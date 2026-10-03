package imageutil

import (
	"bytes"
	"fmt"
)

type resolutionGenerator struct {
	processor ImageTransformer
}

func NewResolutionGenerator(processor ImageTransformer) VariantCreator {
	return &resolutionGenerator{
		processor: processor,
	}
}

func (rG *resolutionGenerator) GenerateVariants(
	data []byte,
	mime MIME,
	specs []VariantSpec,
) ([]GeneratedVariant, error) {
	var variants []GeneratedVariant

	for _, spec := range specs {
		reader := bytes.NewReader(data)

		img, err := rG.processor.Decode(reader, string(mime))
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", spec.Type, err)
		}

		resized := rG.processor.Resize(img, spec.Width)

		finalMIME := mime
		if mime == MIMEWEBP {
			finalMIME = MIMEJPEG
		}

		encoded, err := rG.processor.Encode(resized, string(finalMIME))
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", spec.Type, err)
		}

		variants = append(variants, GeneratedVariant{
			Type:      spec.Type,
			Data:      encoded,
			SizeBytes: int64(len(encoded)),
			MIMEType:  finalMIME,
		})
	}

	return variants, nil
}
