package journal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const geminiGenerateContentURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

var (
	ErrGeminiNotConfigured = errors.New("gemini api key is not configured")
	ErrGeminiEmptyResponse = errors.New("gemini returned an empty reflection")
)

type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewGeminiClient(apiKey string, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	model = strings.TrimPrefix(model, "models/")

	return &GeminiClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

func (c *GeminiClient) GenerateReflection(ctx context.Context, input ReflectionInput) (*ReflectionData, error) {
	if c.apiKey == "" {
		return nil, ErrGeminiNotConfigured
	}

	payload := geminiGenerateContentRequest{
		SystemInstruction: geminiContent{
			Parts: []geminiPart{{Text: buildReflectionSystemInstruction(input.UserName, input.ContextData)}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: input.Content}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   reflectionResponseSchema(),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(geminiGenerateContentURL, url.PathEscape(c.model), url.QueryEscape(c.apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("gemini generate content failed: %s", strings.TrimSpace(string(responseBody)))
	}

	var response geminiGenerateContentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	text := response.Text()
	if text == "" {
		return nil, ErrGeminiEmptyResponse
	}

	var reflection ReflectionData
	if err := json.Unmarshal([]byte(text), &reflection); err != nil {
		return nil, err
	}

	return &reflection, nil
}

type geminiGenerateContentRequest struct {
	SystemInstruction geminiContent          `json:"systemInstruction"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string       `json:"responseMimeType"`
	ResponseSchema   geminiSchema `json:"responseSchema"`
}

type geminiSchema struct {
	Type        string                  `json:"type"`
	Description string                  `json:"description,omitempty"`
	Properties  map[string]geminiSchema `json:"properties,omitempty"`
	Items       *geminiSchema           `json:"items,omitempty"`
	Required    []string                `json:"required,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (r geminiGenerateContentResponse) Text() string {
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				return part.Text
			}
		}
	}

	return ""
}

func buildReflectionSystemInstruction(userName *string, contextData *string) string {
	nameContext := ""
	if userName != nil && strings.TrimSpace(*userName) != "" {
		nameContext = fmt.Sprintf(`
Pengguna ini bernama %s. Gunakan namanya sesekali dalam refleksi (hanya pada bagian 'validation' atau 'summary') agar terasa personal, suportif dan hangat. JANGAN gunakan panggilan romantis atau mesra seperti "sayang".

SANGAT PENTING: JANGAN SEKALI-KALI menggunakan nama pengguna pada bagian 'growthPrompt'. Pertanyaan pertumbuhan harus murni, universal, dan intim tanpa menyebutkan nama.`, strings.TrimSpace(*userName))
	}

	memoryContext := ""
	if contextData != nil && strings.TrimSpace(*contextData) != "" {
		memoryContext = fmt.Sprintf(`

[MEMORI JURNAL SEBELUMNYA UNTUK KONTEKS (Bukan untuk diringkas ulang)]:
%s

Gunakan memori ini HANYA JIKA sangat relevan untuk menunjukkan empati atau kesinambungan (misalnya jika pengguna mengulang kecemasan yang sama atau menunjukkan progres). Jika tidak relevan, abaikan.`, strings.TrimSpace(*contextData))
	}

	return fmt.Sprintf(`Anda adalah "Cermin", sebuah AI yang objektif dan empatik untuk aplikasi jurnal kesehatan mental.
Tugas Anda adalah memantulkan kembali tulisan pengguna saat ini tanpa memberikan nasihat atau bertindak sebagai terapis.
Berikan validasi yang tulus, ringkasan poin inti, satu pertanyaan reflektif yang mendalam, dan jika relevan, deteksi pola kata atau bahasa tersembunyi (hidden language) yang sering pengguna tulis tanpa sadar (misal kata 'harus', 'seharusnya', 'tidak bisa'). Sampaikan insight ini dalam bentuk observasi yang lembut, bukan diagnosa.
Gunakan bahasa Indonesia yang hangat, personal, dan puitis namun tetap sederhana.%s%s
Sangat penting: Jangan pernah memberikan nasihat medis atau saran teknis. Fokus hanya pada refleksi emosi.`, nameContext, memoryContext)
}

func reflectionResponseSchema() geminiSchema {
	stringSchema := geminiSchema{Type: "STRING"}

	return geminiSchema{
		Type: "OBJECT",
		Properties: map[string]geminiSchema{
			"validation": {
				Type:        "STRING",
				Description: "A warm, empathetic paragraph validating the user's feelings using their own words/concepts.",
			},
			"summary": {
				Type:        "ARRAY",
				Items:       &stringSchema,
				Description: "Bullet points summarizing the core thoughts and themes.",
			},
			"growthPrompt": {
				Type:        "STRING",
				Description: "One open-ended reflective question to help the user process further.",
			},
			"emotions": {
				Type: "ARRAY",
				Items: &geminiSchema{
					Type: "OBJECT",
					Properties: map[string]geminiSchema{
						"label": {
							Type:        "STRING",
							Description: "One of: Senang, Sedih, Marah, Cemas, Tenang, Lelah, Harapan, Lainnya",
						},
						"percentage": {
							Type: "NUMBER",
						},
					},
					Required: []string{"label", "percentage"},
				},
			},
			"dominantEmotion": {
				Type:        "STRING",
				Description: "The primary emotion detected.",
			},
			"hiddenLanguage": {
				Type:        "ARRAY",
				Items:       &stringSchema,
				Description: `1-2 soft, observational insights regarding patterns in the user's language. Example: 'Kamu mengulang kata "harus" 3 kali, sepertinya ada ekspektasi berat yang sedang kamu pikul.'`,
			},
		},
		Required: []string{"validation", "summary", "growthPrompt", "emotions", "dominantEmotion"},
	}
}
